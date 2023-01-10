package python

import (
	"path"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/multierr"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type (
	Expose struct{}

	gatewaySpec struct {
		FilePath   string
		AppVarName string
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

	fastapiDefResult struct {
		Expression *sitter.Node
		Identifier *sitter.Node
		RootPath   string
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
		err = core.WrapErrf(err, "express handler failure for %s", unit.Name)
	}

	return err
}

func (h *restAPIHandler) findFastAPIAppDefinition(cap *core.Annotation, f *core.SourceFile) (fastapiDefResult, error) {

	nextMatch := DoQuery(cap.Node, exposeFastAPIAssignment)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		identifier, function, expression, arg, val :=
			match["identifier"], match["function"], match["expression"], match["arg"], match["val"]

		if function.Content(f.Program()) == "FastAPI" {

			rootPath := ""
			if arg != nil && arg.Content(f.Program()) == "root_path" {
				var err error
				rootPath, err = stringLiteralContent(val, f.Program())
				if err != nil {
					return fastapiDefResult{}, errors.Wrap(err, "invalid root_path detected")
				}
				h.log.Sugar().Debugf("Root path detected: %s", rootPath)
			}

			return fastapiDefResult{
				Expression: expression,
				Identifier: identifier,
				RootPath:   rootPath,
			}, nil
		}
	}

	return fastapiDefResult{}, nil
}

func (h *restAPIHandler) handle(unit *core.ExecutionUnit) error {
	h.Unit = unit
	h.log = zap.L().With(zap.String("unit", unit.Name))

	var errs multierr.Error
	for _, f := range unit.Files() {
		src, ok := Language.ID.CastFile(f)
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
		gw := core.NewGateway(spec.AppVarName)
		if existing := h.Result.Get(gw.Key()); existing != nil {
			gw = existing.(*core.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppVarName
			h.Result.Add(gw)
		}

		for _, route := range routes {
			targetKind := ""
			switch h.Unit.Type() {
			// TODO: move these out of expose into runtime somehow
			case "fargate":
				targetKind = core.NetworkLoadBalancerKind
			}

			existsInUnit, it := gw.AddRoute(route.Route, h.Unit, targetKind)
			if existsInUnit != "" {
				h.log.Sugar().Infof("Not adding duplicate route %v for %v. Exists in %v", route.Path, route.ExecUnitName, existsInUnit)
				continue
			}

			targetFileR := unit.Get(route.DefinedInPath)
			targetFile, ok := Language.ID.CastFile(targetFileR)
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

		var appName string
		app, err := h.findFastAPIAppDefinition(capNode, f)
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}
		if app.Expression == nil {
			log.Warn("No listener found")
			continue
		}

		appName = app.Identifier.Content(f.Program())
		h.RootPath = app.RootPath

		gwSpec := gatewaySpec{
			FilePath:   f.Path(),
			AppVarName: cap.ID,
		}

		log = log.With(zap.String("var", appName))

		localRoutes, err := h.findFastAPIRoutesForVar(f, appName, "")
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}

		if len(localRoutes) > 0 {
			log.Sugar().Infof("Found %d route(s) on app '%s'", len(localRoutes), appName)
			h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], localRoutes...)
		}

		// TODO: add support for routers
	}
	return f, nil
}

type routeMethodPath struct {
	Verb string
	Path string
}

func (h *restAPIHandler) findVerbFuncs(root *sitter.Node, source []byte, varName string) ([]routeMethodPath, error) {
	nextMatch := DoQuery(root, exposeVerb)
	var route []routeMethodPath
	var err error
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		appName := match["appName"]
		verb := match["verb"]
		routePath := match["path"]

		if !query.NodeContentEquals(appName, source, varName) {
			continue // wrong var (not the FastAPI app we're looking for)
		}

		if argname := match["argname"]; argname != nil && !query.NodeContentEquals(argname, source, "path") {
			continue // wrong kwarg (i.e. not 'path')
		}

		funcName := verb.Content(source)
		if _, supported := core.Verbs[core.Verb(strings.ToUpper(funcName))]; !supported {
			continue // unsupported verb
		}

		var pathContent string
		pathContent, err = stringLiteralContent(routePath, source)
		if err != nil {
			return []routeMethodPath{}, errors.Wrap(err, "invalid verb path")
		}

		route = append(route, routeMethodPath{
			Verb: verb.Content(source),
			Path: pathContent,
		})
	}
	return route, err
}

// findFastAPIRoutesForVar finds any routes defined on varName declared in core.SourceFile f
func (h *restAPIHandler) findFastAPIRoutesForVar(f *core.SourceFile, varName string, prefix string) ([]gatewayRouteDefinition, error) {
	// TODO add support for finding additional routes that may have been added in files where this varName has been imported
	var routes = make([]gatewayRouteDefinition, 0)
	log := h.log.With(logging.FileField(f))

	verbFuncs, err := h.findVerbFuncs(f.Tree().RootNode(), f.Program(), varName)

	log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	for _, vfunc := range verbFuncs {
		route := core.Route{
			Verb:          core.Verb(vfunc.Verb),
			Path:          sanitizeFastapiPath(path.Join(h.RootPath, prefix, vfunc.Path)),
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

var fastapiPathParamPattern = regexp.MustCompile(`{([\w*]+)}`)

// sanitizeFastapiPath converts fastapi path parameters to Express syntax,
// but does not perform validation to ensure that the supplied string is a valid fastapi route.
// As such, there's no expectation of correct output for invalid paths
func sanitizeFastapiPath(path string) string {
	sanitized := strings.ReplaceAll(path, ":path}", "*}")
	return fastapiPathParamPattern.ReplaceAllString(sanitized, ":$1")
}
