package python

import (
	"path"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/multierr"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/construct"
	klotho_errors "github.com/klothoplatform/klotho/pkg/errors"
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
		gatewayId  string
	}

	gatewayRouteDefinition struct {
		types.Route
		DefinedInPath string
	}

	restAPIHandler struct {
		ConstructGraph  *construct.ConstructGraph
		Unit            *types.ExecutionUnit
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

func (p Expose) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {
	var errs multierr.Error
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		err := p.transformSingle(constructGraph, unit)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (p *Expose) transformSingle(constructGraph *construct.ConstructGraph, unit *types.ExecutionUnit) error {
	h := &restAPIHandler{ConstructGraph: constructGraph, RoutesByGateway: make(map[gatewaySpec][]gatewayRouteDefinition)}
	err := h.handle(unit)
	if err != nil {
		err = klotho_errors.WrapErrf(err, "express handler failure for %s", unit.Name)
	}

	return err
}

func (h *restAPIHandler) findFastAPIAppDefinition(cap *types.Annotation, f *types.SourceFile) (fastapiDefResult, error) {

	nextMatch := DoQuery(cap.Node, exposeFastAPIAssignment)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		identifier, function, expression, arg, val :=
			match["identifier"], match["function"], match["expression"], match["arg"], match["val"]

		if function.Content() == "FastAPI" {

			rootPath := ""
			if arg != nil && arg.Content() == "root_path" {
				var err error
				rootPath, err = stringLiteralContent(val)
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

func (h *restAPIHandler) handle(unit *types.ExecutionUnit) error {
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
		gw := types.NewGateway(spec.gatewayId)
		if existing := h.ConstructGraph.GetConstruct(gw.Id()); existing != nil {
			gw = existing.(*types.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppVarName
			h.ConstructGraph.AddConstruct(gw)
		}

		for _, route := range routes {
			existsInUnit := gw.AddRoute(route.Route, h.Unit)
			if existsInUnit != "" {
				h.log.Sugar().Infof("Not adding duplicate route %v for %v. Exists in %v", route.Path, route.ExecUnitName, existsInUnit)
				continue
			}

			targetFileR := unit.Get(route.DefinedInPath)
			targetFile, ok := Language.ID.CastFile(targetFileR)
			if !ok {
				continue
			}

			targetUnit := types.FileExecUnitName(targetFile)
			if targetUnit == "" {
				// if the target file is in all units, direct the API gateway to use the unit that defines the listener
				targetUnit = unit.Name
			}
			h.ConstructGraph.AddDependency(gw.Id(), construct.ResourceId{
				Provider: construct.AbstractConstructProvider,
				Type:     annotation.ExecutionUnitCapability,
				Name:     targetUnit,
			})
		}
	}

	return errs.ErrOrNil()
}

func (h *restAPIHandler) handleFile(f *types.SourceFile) (*types.SourceFile, error) {
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
			return nil, types.NewCompilerError(f, capNode,
				errors.New("expose capability must specify target = \"public\""))

		}

		var appVarName string
		app, err := h.findFastAPIAppDefinition(capNode, f)
		if err != nil {
			return nil, types.NewCompilerError(f, capNode, err)
		}
		if app.Expression == nil {
			log.Warn("No listener found")
			continue
		}

		appVarName = app.Identifier.Content()
		h.RootPath = app.RootPath

		gwSpec := gatewaySpec{
			FilePath:   f.Path(),
			AppVarName: appVarName,
			gatewayId:  cap.ID,
		}

		log = log.With(zap.String("var", appVarName))

		localRoutes, err := h.findFastAPIRoutesForVar(f, appVarName, "")
		if err != nil {
			return nil, types.NewCompilerError(f, capNode, err)
		}

		if len(localRoutes) > 0 {
			log.Sugar().Infof("Found %d route(s) on app '%s'", len(localRoutes), appVarName)
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

func (h *restAPIHandler) findVerbFuncs(root *sitter.Node, varName string) ([]routeMethodPath, error) {
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

		if !query.NodeContentEquals(appName, varName) {
			continue // wrong var (not the FastAPI app we're looking for)
		}

		if argname := match["argname"]; argname != nil && !query.NodeContentEquals(argname, "path") {
			continue // wrong kwarg (i.e. not 'path')
		}

		funcName := verb.Content()
		if _, supported := types.Verbs[types.Verb(strings.ToUpper(funcName))]; !supported {
			continue // unsupported verb
		}

		var pathContent string
		pathContent, err = stringLiteralContent(routePath)
		if err != nil {
			return []routeMethodPath{}, errors.Wrap(err, "invalid verb path")
		}

		route = append(route, routeMethodPath{
			Verb: verb.Content(),
			Path: pathContent,
		})
	}
	return route, err
}

// findFastAPIRoutesForVar finds any routes defined on varName declared in types.SourceFile f
func (h *restAPIHandler) findFastAPIRoutesForVar(f *types.SourceFile, varName string, prefix string) ([]gatewayRouteDefinition, error) {
	// TODO add support for finding additional routes that may have been added in files where this varName has been imported
	var routes = make([]gatewayRouteDefinition, 0)
	log := h.log.With(logging.FileField(f))

	verbFuncs, err := h.findVerbFuncs(f.Tree().RootNode(), varName)

	log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	for _, vfunc := range verbFuncs {
		route := types.Route{
			Verb:          types.Verb(vfunc.Verb),
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
