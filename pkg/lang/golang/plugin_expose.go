package golang

import (
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type (
	Expose struct{}

	gatewaySpec struct {
		FilePath   string
		AppVarName string
		Id         string
	}

	gatewayRouteDefinition struct {
		core.Route
		DefinedInPath string
	}

	restAPIHandler struct {
		Result          *core.CompilationResult
		Deps            *core.Dependencies
		Unit            *core.ExecutionUnit
		RoutesByGateway map[gatewaySpec][]gatewayRouteDefinition
		RootPath        string
		log             *zap.Logger
	}

	chiRouterDefResult struct {
		Declaration *sitter.Node
		Identifier  *sitter.Node
		RootPath    string
	}

	httpListener struct {
		Identifier *sitter.Node
		Expression *sitter.Node
		Address    *sitter.Node
	}

	routeMethodPath struct {
		Verb string
		Path string
	}
)

func (p *Expose) Name() string { return "Expose" }

func (p Expose) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		err := p.transformSingle(result, deps, unit)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (p *Expose) transformSingle(result *core.CompilationResult, deps *core.Dependencies, unit *core.ExecutionUnit) error {
	h := &restAPIHandler{Result: result, Deps: deps, RoutesByGateway: make(map[gatewaySpec][]gatewayRouteDefinition)}
	err := h.handle(unit)
	if err != nil {
		err = core.WrapErrf(err, "Chi handler failure for %s", unit.Name)
	}
	return err
}

func (h *restAPIHandler) handle(unit *core.ExecutionUnit) error {
	h.Unit = unit
	h.log = zap.L().With(zap.String("unit", unit.Name))

	var errs multierr.Error
	for _, f := range unit.Files() {
		src, ok := goLang.CastFile(f)
		if !ok {
			continue
		}

		newF, err := h.handleFile(src)
		if err != nil {
			errs.Append(err)
			continue
		}
		if newF != nil {
			unit.Add(newF)
		}
	}

	for spec, routes := range h.RoutesByGateway {
		gwName := spec.Id
		gw := core.NewGateway(gwName)
		if existing := h.Result.Get(gw.Key()); existing != nil {
			gw = existing.(*core.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppVarName
			h.Result.Add(gw)
		}

		for _, route := range routes {
			existsInUnit, it := gw.AddRoute(route.Route, h.Unit, "")
			if existsInUnit != "" {
				h.log.Sugar().Infof("Not adding duplicate route %v for %v. Exists in %v", route.Path, route.ExecUnitName, existsInUnit)
				continue
			}

			targetFileR := unit.Get(route.DefinedInPath)
			targetFile, ok := goLang.CastFile(targetFileR)
			if !ok {
				continue
			}

			targetUnit := core.FileExecUnitName(targetFile)
			if targetUnit == "" {
				// if the target file is in all units, direct the API gateway to use the unit that defines the listener
				targetUnit = unit.Name
			}
			if it.ExecUnitName == "" {
				h.Deps.Add(gw.Key(), core.ResourceKey{Name: targetUnit, Kind: core.ExecutionUnitKind})
			} else {
				// If an integration target exists for an exec unit, create the cloud resource and set the deps as gw -> it -> route exec unit
				if existing := h.Result.Get(it.Key()); existing == nil {
					h.Result.Add(it)
				}
				h.Deps.Add(gw.Key(), it.Key())
				h.Deps.Add(it.Key(), core.ResourceKey{Name: targetUnit, Kind: core.ExecutionUnitKind})
			}
		}
	}

	return errs.ErrOrNil()
}

func (h *restAPIHandler) handleFile(f *core.SourceFile) (*core.SourceFile, error) {

	caps := f.Annotations()
	for _, capNode := range caps {
		log := zap.L().With(logging.AnnotationField(capNode), logging.FileField(f))
		cap := capNode.Capability
		if cap.Name != annotation.ExposeCapability {
			continue
		}

		// target can be public or private for now
		// currently private is unimplemented, so
		// we fail unless it's set to public
		target, ok := cap.Directives.String("target")
		if !ok {
			target = "private"
		}
		if target != "public" {
			return nil, core.NewCompilerError(f, capNode,
				errors.New("expose capability must specify target = \"public\""))

		}

		listener, err := h.findHttpListenAndServe(capNode, f)
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}
		if listener.Expression == nil {
			log.Warn("No http listen found")
			continue
		}
		appName := listener.Identifier.Content(f.Program())

		importsNode, err := h.FindImports(f)
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}

		//TODO: Move comment listen code to library logic like JS does eventually
		if h.Unit.ExecType == "lambda" {
			//TODO: Will likely need to move this into a separate plugin of some sort
			// Instead of having a dispatcher file, the dipatcher logic is injected into the main.go file. By having that
			// logic in the expose plugin though, it will only happen if they use the expose annotation for the lambda case.
			updatedListenContent := UpdateListenWithHandlerCode(string(f.Program()), listener.Expression.Content(f.Program()), appName)

			updatedImportContent := UpdateImportWithHandlerRequirements(updatedListenContent, importsNode, f)

			err = UpdateGoModWithHandlerRequirements(h.Unit)
			if err != nil {
				return f, errors.Wrap(err, "error updating imports for handler")
			}

			err := f.Reparse([]byte(updatedImportContent))
			if err != nil {
				return f, errors.Wrap(err, "error reparsing after substitutions")
			}
		}

		router, err := h.findChiRouterDefinition(f, appName)
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}
		if router.Declaration == nil {
			log.Warn("No Router found")
			continue
		}

		h.RootPath = router.RootPath

		gwSpec := gatewaySpec{
			FilePath:   f.Path(),
			AppVarName: appName,
			Id:         cap.ID,
		}

		log = log.With(zap.String("var", appName))

		localRoutes, err := h.findChiRoutesForVar(f, appName, "")
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}

		if len(localRoutes) > 0 {
			log.Sugar().Infof("Found %d route(s) on app '%s'", len(localRoutes), appName)
			h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], localRoutes...)
		}
	}
	return f, nil
}

func (h *restAPIHandler) findChiRouterDefinition(f *core.SourceFile, appName string) (chiRouterDefResult, error) {
	nextMatch := doQuery(f.Tree().RootNode(), findRouterAssignment)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		identifier, definition, declaration := match["identifier"], match["definition"], match["declaration"]

		if definition.Content(f.Program()) == "chi.NewRouter()" {
			foundName := identifier.Content(f.Program())
			if foundName == appName {
				rootPath := ""
				return chiRouterDefResult{
					Declaration: declaration,
					Identifier:  identifier,
					RootPath:    rootPath,
				}, nil
			} else {
				return chiRouterDefResult{}, errors.Errorf("Invalid router assignment: Expected [%s] actual [%s]", appName, foundName)
			}
		}
	}

	return chiRouterDefResult{}, nil
}

func (h *restAPIHandler) findHttpListenAndServe(cap *core.Annotation, f *core.SourceFile) (httpListener, error) {
	nextMatch := doQuery(cap.Node, findHttpListen)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		listenExp, addr, router, expression := match["sel_exp"], match["addr"], match["router"], match["expression"]

		if listenExp.Content(f.Program()) == "http.ListenAndServe" {
			return httpListener{
				Identifier: router,
				Expression: expression,
				Address:    addr,
			}, nil
		} else {
			return httpListener{}, errors.Errorf("Expected http.ListenAndServe but found %s", listenExp.Content(f.Program()))
		}
	}

	return httpListener{}, nil
}

func (h *restAPIHandler) findChiRoutesForVar(f *core.SourceFile, varName string, prefix string) ([]gatewayRouteDefinition, error) {
	var routes = make([]gatewayRouteDefinition, 0)
	log := h.log.With(logging.FileField(f))

	//TODO: This is looking for routes defined in the file. In the multi exec case will need to look for 'Mount' as well.
	verbFuncs, err := h.findVerbFuncs(f.Tree().RootNode(), f.Program(), varName)
	if err != nil {
		return routes, err
	}

	log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	for _, vfunc := range verbFuncs {
		route := core.Route{
			Verb:          core.Verb(vfunc.Verb),
			Path:          path.Join(h.RootPath, prefix, vfunc.Path), //TODO: Handle Chi router path parameters conversion to express for pulumi logic
			ExecUnitName:  h.Unit.Name,
			HandledInFile: f.Path(),
		}
		log.Sugar().Debugf("Found route function %s %s for '%s'", route.Verb, route.Path, varName)
		routes = append(routes, gatewayRouteDefinition{
			Route:         route,
			DefinedInPath: f.Path(),
		})
	}
	return routes, err
}

func (h *restAPIHandler) findVerbFuncs(root *sitter.Node, source []byte, varName string) ([]routeMethodPath, error) {
	nextMatch := doQuery(root, findExposeVerb)
	var route []routeMethodPath
	var err error
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		appName := match["routerName"]
		verb := match["verb"]
		routePath := match["path"]

		if !query.NodeContentEquals(appName, source, varName) {
			continue // wrong var (not the Chi router we're looking for)
		}

		funcName := verb.Content(source)

		if _, supported := core.Verbs[core.Verb(strings.ToUpper(funcName))]; !supported {
			continue // unsupported verb
		}

		pathContent := stringLiteralContent(routePath, source)

		route = append(route, routeMethodPath{
			Verb: verb.Content(source),
			Path: pathContent,
		})
	}
	return route, err
}

func (h *restAPIHandler) FindImports(f *core.SourceFile) (*sitter.Node, error) {
	nextMatch := doQuery(f.Tree().RootNode(), findImports)
	var err error
	var imports *sitter.Node
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		imports := match["expression"]

		if imports != nil {
			return imports, nil
		}
	}
	return imports, err
}
