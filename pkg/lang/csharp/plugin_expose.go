package csharp

import (
	"fmt"
	"path"
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
		FilePath       string
		AppBuilderName string
		gatewayId      string
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

// An exposeUseEndpointsResult represents an ASP.net Core IApplicationBuilder.UseEndpoints() invocation
type exposeUseEndpointsResult struct {
	UseExpression                  *sitter.Node // Expression of the UseEndpoints() invocation (app.UseEndpoints(endpoints => {...})
	AppBuilderIdentifier           *sitter.Node // Identifier of the builder (IApplicationBuilder app)
	EndpointRouteBuilderIdentifier *sitter.Node // Identifier of the RoutesBuilder param (endpoints => {...})
}

func findIApplicationBuilder(cap *core.Annotation) []exposeUseEndpointsResult {
	var results []exposeUseEndpointsResult
	nextMatch := DoQuery(query.FirstAncestorOfType(cap.Node, "namespace_declaration"), configuredApp)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		paramNameN := match["param_name"]

		if paramNameN == nil {
			break
		}

		paramName := paramNameN.Content()

		nextExpressionMatch := DoQuery(query.FirstAncestorOfType(paramNameN, "method_declaration"), fmt.Sprintf(useEndpointsFormat, paramName))
		for {
			match, found := nextExpressionMatch()
			if !found {
				break
			}

			results = append(results, exposeUseEndpointsResult{
				UseExpression:                  match["expression"],
				AppBuilderIdentifier:           paramNameN,
				EndpointRouteBuilderIdentifier: match["endpoints_param"],
			})
		}
	}

	return results
}

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
		gw := core.NewGateway(spec.gatewayId)
		if existing := h.Result.Get(gw.Key()); existing != nil {
			gw = existing.(*core.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppBuilderName
			h.Result.Add(gw)
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

			targetUnit := core.FileExecUnitName(targetFile)
			if targetUnit == "" {
				// if the target file is in all units, direct the API gateway to use the unit that defines the listener
				targetUnit = unit.Name
			}
			h.Deps.Add(gw.Key(), core.ResourceKey{Name: targetUnit, Kind: core.ExecutionUnitKind})
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

		useEndpointResults := findIApplicationBuilder(capNode)

		if useEndpointResults == nil {
			log.Warn(`No "IApplicationBuilder.UseEndpoint()" invocations found`)
			continue
		}

		for _, useEndpoint := range useEndpointResults {
			var appBuilderName string = useEndpoint.EndpointRouteBuilderIdentifier.Content()
			var endpointRouteBuilderName string = useEndpoint.EndpointRouteBuilderIdentifier.Content()

			gwSpec := gatewaySpec{
				FilePath:       f.Path(),
				AppBuilderName: appBuilderName,
				gatewayId:      cap.ID,
			}

			log = log.With(
				zap.String("IApplicationBuilder", appBuilderName),
				zap.String("IEndpointRouteBuilder", endpointRouteBuilderName),
			)

			localRoutes, err := h.findLocallyMappedRoutes(f, endpointRouteBuilderName, "")
			if err != nil {
				return nil, core.NewCompilerError(f, capNode, err)
			}

			if len(localRoutes) > 0 {
				log.Sugar().Infof("Found %d route(s) on app '%s'", len(localRoutes), appBuilderName)
				h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], localRoutes...)
			}

			// TODO: add support for decorated controllers
		}
	}
	return f, nil
}

type routeMethodPath struct {
	Verb string
	Path string
}

func (h *restAPIHandler) findVerbMappings(root *sitter.Node, varName string) ([]routeMethodPath, error) {
	nextMatch := DoQuery(root, exposeMapRoute)
	var route []routeMethodPath
	var err error
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		invokedVarName := match["var"]
		methodName := match["method_name"]
		routePath := match["path"]

		if !query.NodeContentEquals(invokedVarName, varName) {
			continue // wrong var (not the IApplicationBuilder we're looking for)
		}

		verb := strings.ToUpper(strings.TrimPrefix(methodName.Content(), "Map"))
		if verb == "" {
			verb = "ANY"
		}

		if _, supported := core.Verbs[core.Verb(verb)]; !supported {
			continue // unsupported verb
		}

		route = append(route, routeMethodPath{
			Verb: verb,
			Path: stringLiteralContent(routePath),
		})
	}
	return route, err
}

// findLocallyMappedRoutes finds any routes defined on varName declared in core.SourceFile f
func (h *restAPIHandler) findLocallyMappedRoutes(f *core.SourceFile, varName string, prefix string) ([]gatewayRouteDefinition, error) {
	var routes = make([]gatewayRouteDefinition, 0)
	log := h.log.With(logging.FileField(f))

	verbFuncs, err := h.findVerbMappings(f.Tree().RootNode(), varName)

	log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	for _, vfunc := range verbFuncs {
		route := core.Route{
			Verb:          core.Verb(vfunc.Verb),
			Path:          sanitizeAspDotNetPath(path.Join(h.RootPath, prefix, vfunc.Path)),
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

// sanitizeAspDotNetPath converts ASP.net path parameters to Express syntax,
// but does not perform validation to ensure that the supplied string is a valid ASP.net route.
// As such, there's no expectation of correct output for invalid paths
func sanitizeAspDotNetPath(path string) string {
	// TODO: implement path param sanitization
	return path
}
