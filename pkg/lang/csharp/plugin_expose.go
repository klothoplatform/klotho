package csharp

import (
	"fmt"
	"path"
	"strings"

	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"

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
	Expose      struct{}
	gatewaySpec struct {
		FilePath        string // The path of the file containing the expose annotation
		AppBuilderName  string // Name of the argument that UseEndpoints() is invoked on
		MapsControllers bool   // Flag indicating that controller detection should be enabled for this spec
		gatewayId       string
	}
	gatewayRouteDefinition struct {
		core.Route
		DefinedInPath string
	}
	aspDotNetCoreHandler struct {
		ConstructGraph   *core.ConstructGraph
		Unit             *core.ExecutionUnit
		RoutesByGateway  map[gatewaySpec][]gatewayRouteDefinition
		RootPath         string
		log              *zap.Logger
		ControllerRoutes []gatewayRouteDefinition
	}

	// actionSpec represents a fully parsed ASP.NET Core action method
	actionSpec struct {
		method           MethodDeclaration
		verb             core.Verb
		routeTemplate    string
		name             string
		hasRouteTemplate bool // an empty string is a valid routeTemplate value
	}

	// controllerSpec represents a fully parsed ASP.NET Core controller (including its actions)
	controllerSpec struct {
		execUnitName string
		name         string
		class        TypeDeclaration
		actions      []actionSpec
		controllerAttributeSpec
	}

	// controllerAttributeSpec represents the details derived from
	// routing attributes applied to an ASP.NET Core Controller class
	controllerAttributeSpec struct {
		routeTemplates []string
		area           string
	}

	// useEndpointsResult represents an ASP.Net Core IApplicationBuilder.UseEndpoints() invocation
	useEndpointsResult struct {
		StartupClass                   ASPDotNetCoreStartupClass // Declaration of the Startup class surrounding the expose annotation
		UseExpression                  *sitter.Node              // Expression of the UseEndpoints() invocation (app.UseEndpoints(endpoints => {...})
		AppBuilderIdentifier           *sitter.Node              // Identifier of the builder (IApplicationBuilder app)
		EndpointRouteBuilderIdentifier *sitter.Node              // Identifier of the RoutesBuilder param (endpoints => {...})
	}

	// routeMethodPath is a simple mapping between an HTTP Verb and a path
	routeMethodPath struct {
		Verb core.Verb
		Path string
	}
)

func (p *Expose) Name() string { return "Expose" }

func (p *Expose) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error

	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](constructGraph) {
		err := p.transformSingle(constructGraph, unit)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (p *Expose) transformSingle(constructGraph *core.ConstructGraph, unit *core.ExecutionUnit) error {
	h := &aspDotNetCoreHandler{
		ConstructGraph:  constructGraph,
		RoutesByGateway: make(map[gatewaySpec][]gatewayRouteDefinition),
	}
	err := h.handle(unit)
	if err != nil {
		err = core.WrapErrf(err, "ASP.NET Core handler failed for %s", unit.ID)
	}

	return err
}

func (h *aspDotNetCoreHandler) handle(unit *core.ExecutionUnit) error {
	h.Unit = unit
	h.log = zap.L().With(zap.String("unit", unit.ID))

	var errs multierr.Error
	for _, f := range unit.FilesOfLang(CSharp) {
		newF, err := h.handleFile(f)
		if err != nil {
			errs.Append(err)
			continue
		}
		if newF != nil {
			unit.Add(newF)
		}
	}

	for spec, routes := range h.RoutesByGateway {
		gw := core.NewGateway(core.AnnotationKey{ID: spec.gatewayId, Capability: annotation.ExposeCapability})
		if existing := h.ConstructGraph.GetConstruct(gw.Id()); existing != nil {
			gw = existing.(*core.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppBuilderName
			h.ConstructGraph.AddConstruct(gw)
		}

		if spec.MapsControllers {
			routes = append(routes, h.ControllerRoutes...)
			if len(h.ControllerRoutes) > 0 {
				h.log.Sugar().Debugf("Adding controller routes to gateway %+v", spec)
			}
		}

		zap.L().Sugar().Infof("Found %d route(s) on app '%s'", len(routes), spec.AppBuilderName)

		if len(routes) == 0 && len(gw.Routes) == 0 {
			h.log.Sugar().Infof("Adding catchall route for gateway %+v with no detected routes", spec)
			routes = []gatewayRouteDefinition{
				{
					Route: core.Route{
						Path:          "/",
						ExecUnitName:  unit.ID,
						Verb:          core.VerbAny,
						HandledInFile: spec.FilePath,
					},
					DefinedInPath: spec.FilePath,
				},
				{
					Route: core.Route{
						Path:          "/:proxy*",
						ExecUnitName:  unit.ID,
						Verb:          core.VerbAny,
						HandledInFile: spec.FilePath,
					},
					DefinedInPath: spec.FilePath,
				},
			}
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
				targetUnit = unit.ID
			}
			h.ConstructGraph.AddDependency(gw.Id(), core.ResourceId{
				Provider: core.AbstractConstructProvider,
				Type:     annotation.ExecutionUnitCapability,
				Name:     targetUnit,
			})
		}
	}

	return errs.ErrOrNil()
}

func (h *aspDotNetCoreHandler) handleFile(f *core.SourceFile) (*core.SourceFile, error) {
	caps := f.Annotations()

	for _, capAnnotation := range caps {
		capability := capAnnotation.Capability
		if capability.Name != annotation.ExposeCapability {
			continue
		}

		// target can be public or private for now
		// currently private is unimplemented, so
		// we fail unless it's set to public
		target, ok := capability.Directives.String("target")
		if !ok {
			target = "private"
		}
		if target != "public" {
			return nil, core.NewCompilerError(f, capAnnotation,
				errors.New("expose capability must specify target = \"public\""))

		}

		useEndpointResults := findIApplicationBuilder(capAnnotation)

		if useEndpointResults == nil {
			h.log.With(logging.NodeField(capAnnotation.Node)).Warn(`No "IApplicationBuilder.UseEndpoint()" invocations found`)
			continue
		}

		for _, useEndpoint := range useEndpointResults {
			var appBuilderName = useEndpoint.AppBuilderIdentifier.Content()
			var endpointRouteBuilderName = useEndpoint.EndpointRouteBuilderIdentifier.Content()

			gwSpec := gatewaySpec{
				FilePath:        f.Path(),
				AppBuilderName:  appBuilderName,
				gatewayId:       capability.ID,
				MapsControllers: isMapControllersInvoked(useEndpoint) && areControllersInjected(useEndpoint.StartupClass.Class.Node),
			}
			h.RoutesByGateway[gwSpec] = []gatewayRouteDefinition{}

			localRoutes, err := h.findLocallyMappedRoutes(f, endpointRouteBuilderName, "")
			if err != nil {
				return nil, core.NewCompilerError(f, capAnnotation, err)
			}

			h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], localRoutes...)
		}
	}

	controllers := h.findControllersInFile(f)
	for _, c := range controllers {
		for _, route := range c.resolveRoutes() {
			h.ControllerRoutes = append(h.ControllerRoutes, route)
			zap.L().Sugar().Debugf("Mapped route %s %s from %s", route.Verb, route.Path, c.class.Name)
		}
	}

	return f, nil
}

func findIApplicationBuilder(cap *core.Annotation) []useEndpointsResult {
	var results []useEndpointsResult
	classNode := query.FirstAncestorOfType(cap.Node, "class_declaration")
	classDeclaration, ok := getDotnetCoreStartupClass(classNode)
	if !ok {
		return nil
	}

	nextMatch := DoQuery(classNode, exposeStartupConfigure)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		if !IsValidTypeName(match["param1_type"], "Microsoft.AspNetCore.Builder", "IApplicationBuilder") ||
			!IsValidTypeName(match["param2_type"], "Microsoft.AspNetCore.Hosting", "IWebHostEnvironment") {
			continue
		}

		paramNameN := match["param_name"]

		if paramNameN == nil {
			break
		}

		paramName := paramNameN.Content()

		nextExpressionMatch := DoQuery(query.FirstAncestorOfType(paramNameN, "method_declaration"), fmt.Sprintf(useEndpointsFormat, paramName))
		for {
			expressionMatch, found := nextExpressionMatch()
			if !found {
				break
			}

			results = append(results, useEndpointsResult{
				UseExpression:                  expressionMatch["expression"],
				AppBuilderIdentifier:           paramNameN,
				EndpointRouteBuilderIdentifier: expressionMatch["endpoints_param"],
				StartupClass:                   classDeclaration,
			})
		}
	}

	return results
}

func isMapControllersInvoked(useEndpoints useEndpointsResult) bool {
	_, found := DoQuery(useEndpoints.UseExpression, fmt.Sprintf(exposeMapControllersFormat, useEndpoints.EndpointRouteBuilderIdentifier.Content()))()
	return found
}

func (h *aspDotNetCoreHandler) findVerbMappings(root *sitter.Node, varName string) ([]routeMethodPath, error) {
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

		verb := core.Verb(strings.ToUpper(strings.TrimPrefix(methodName.Content(), "Map")))
		if verb == "" {
			verb = core.VerbAny
		}

		if _, supported := core.Verbs[core.Verb(verb)]; !supported {
			continue // unsupported verb
		}

		route = append(route, routeMethodPath{
			Verb: verb,
			Path: normalizedStringContent(routePath),
		})
	}
	return route, err
}

// areControllersInjected evaluates if the current startup class injects controllers into its DI service collection
func areControllersInjected(startupClass *sitter.Node) bool {

	match, found := DoQuery(startupClass, exposeStartupConfigureServices)()
	if !found {
		return false
	}

	if !IsValidTypeName(match["param_type"], "Microsoft.Extensions.DependencyInjection", "IServiceCollection") {
		return false
	}

	methodDeclaration := match["method_declaration"]
	paramName := match["param_name"].Content()

	_, found = DoQuery(methodDeclaration.ChildByFieldName("body"), fmt.Sprintf(exposeAddControllersFormat, paramName))()
	return found

}

// findLocallyMappedRoutes finds any routes defined on varName declared in core.SourceFile f
func (h *aspDotNetCoreHandler) findLocallyMappedRoutes(f *core.SourceFile, varName string, prefix string) ([]gatewayRouteDefinition, error) {
	var routes = make([]gatewayRouteDefinition, 0)

	verbFuncs, err := h.findVerbMappings(f.Tree().RootNode(), varName)

	h.log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	for _, vfunc := range verbFuncs {

		// add fixed route for second to last segment if last path segment is an optional param
		// e.g. /a/{b?} -> /a, /a/:b
		nonOptionalRoute := stripOptionalLastSegment(vfunc.Path)
		if nonOptionalRoute != vfunc.Path {
			routes = append(routes, gatewayRouteDefinition{
				Route: core.Route{
					Verb:          vfunc.Verb,
					Path:          strings.ToLower(sanitizeConventionalPath(path.Join("/", h.RootPath, prefix, nonOptionalRoute))),
					ExecUnitName:  h.Unit.ID,
					HandledInFile: f.Path(),
				},
				DefinedInPath: f.Path(),
			})
		}

		route := core.Route{
			Verb: vfunc.Verb,
			// using lowercase routes enforces consistency
			// since ASP.NET core is case-insensitive and API Gateway is case-sensitive
			Path:          strings.ToLower(sanitizeConventionalPath(path.Join("/", h.RootPath, prefix, vfunc.Path))),
			ExecUnitName:  h.Unit.ID,
			HandledInFile: f.Path(),
		}
		h.log.Sugar().Debugf("Found route function %s %s for '%s'", route.Verb, route.Path, varName)
		routes = append(routes, gatewayRouteDefinition{
			Route:         route,
			DefinedInPath: f.Path(),
		})
	}
	return routes, err
}

// findControllersInFile returns the specs for each controller found in the supplied file
// controller docs: https://learn.microsoft.com/en-us/aspnet/core/mvc/controllers/actions?view=aspnetcore-7.0
func (h *aspDotNetCoreHandler) findControllersInFile(file *core.SourceFile) []controllerSpec {
	types := FindDeclarationsInFile[*TypeDeclaration](file).Declarations()
	controllers := filter.NewSimpleFilter(
		predicate.AnyOf(
			HasBaseWithSuffix("Controller"),
			NameHasSuffix[*TypeDeclaration]("Controller"),
			HasAttribute[*TypeDeclaration]("Microsoft.AspNetCore.Mvc.Controller"),
			HasAttribute[*TypeDeclaration]("Microsoft.AspNetCore.Mvc.ApiController"),
		),
		predicate.Not(HasAttribute[*TypeDeclaration]("Microsoft.AspNetCore.Mvc.NonController")),
	).Apply(types...)
	var controllerSpecs []controllerSpec
	for _, c := range controllers {
		controller := *c
		spec := controllerSpec{
			name:                    strings.TrimSuffix(controller.Name, "Controller"),
			class:                   controller,
			controllerAttributeSpec: parseControllerAttributes(controller),
			actions:                 findActionsInController(controller),
			execUnitName:            h.Unit.ID,
		}
		controllerSpecs = append(controllerSpecs, spec)
	}
	return controllerSpecs
}

// resolveRoutes returns a list of gatewayRouteDefinitions
// by merging a controller's annotations with those of its contained actions
func (c controllerSpec) resolveRoutes() []gatewayRouteDefinition {
	var routes []gatewayRouteDefinition

	prefixes := c.routeTemplates
	hasPrefixes := true
	if len(c.routeTemplates) == 0 {
		prefixes = []string{""}
		hasPrefixes = false
	}

	for _, action := range c.actions {
		for _, prefix := range prefixes {
			routeTemplate := action.routeTemplate
			// action route templates starting with "/" or "~/" indicate controller-level route prefixes should be ignored
			if strings.HasPrefix(routeTemplate, "~/") {
				routeTemplate = strings.TrimPrefix(routeTemplate, "~")
			} else if !strings.HasPrefix(routeTemplate, "/") {
				routeTemplate = path.Join("/", prefix, action.routeTemplate)
			}

			if !hasPrefixes && !action.hasRouteTemplate {
				zap.L().Sugar().Debugf("%s cannot be mapped to a route: no route template found", action.method.QualifiedName)
				continue
			}

			verb := action.verb
			if verb == "" {
				verb = core.VerbAny
			}

			// add fixed route for second to last segment if last routeTemplate segment is an optional param
			// e.g. /a/{b?} -> /a, /a/:b
			nonOptionalRoute := stripOptionalLastSegment(routeTemplate)
			if nonOptionalRoute != routeTemplate {
				routes = append(routes, gatewayRouteDefinition{
					Route: core.Route{
						Verb: verb,
						Path: strings.ToLower(
							sanitizeAttributeBasedPath(nonOptionalRoute, c.area, c.name, action.name)),
						ExecUnitName:  c.execUnitName,
						HandledInFile: c.class.DeclaringFile,
					},
					DefinedInPath: c.class.DeclaringFile,
				})
			}

			routes = append(routes, gatewayRouteDefinition{
				Route: core.Route{
					Verb: verb,
					// using lowercase routes enforces consistency
					// since ASP.NET Core is case-insensitive and API Gateway is case-sensitive
					Path:          strings.ToLower(sanitizeAttributeBasedPath(routeTemplate, c.area, c.name, action.name)),
					ExecUnitName:  c.execUnitName,
					HandledInFile: c.class.DeclaringFile,
				},
				DefinedInPath: c.class.DeclaringFile,
			},
			)
		}
	}
	return routes
}

// findActionsInController returns a []actionSpec containing specifications for each action detected in the supplied controller
// action routing docs: https://learn.microsoft.com/en-us/aspnet/core/mvc/controllers/routing?view=aspnetcore-7.0#ar6
func findActionsInController(controller TypeDeclaration) []actionSpec {
	var actions []actionSpec
	methods := FindDeclarationsAtNode[*MethodDeclaration](controller.Node).Declarations()
	for _, m := range methods {
		if _, isNonAction := m.Attributes()["NonAction"]; !isNonAction {
			actions = append(actions, parseActionAttributes(*m)...)
		}
	}
	return actions
}

// parseControllerAttributes returns a list of controllerAttributeSpecs containing routing information
// for the supplied controller based on its applied attributes
func parseControllerAttributes(controller TypeDeclaration) controllerAttributeSpec {
	attrs := controller.Attributes().OfType("Microsoft.AspNetCore.Mvc.Route", "Microsoft.AspNetCore.Mvc.Area")
	attrSpec := controllerAttributeSpec{}
	for _, attr := range attrs {
		args := attr.Args()
		_, name := splitQualifiedName(attr.Name)
		switch name {
		case "Route":
			if len(args) > 0 {
				attrSpec.routeTemplates = append(attrSpec.routeTemplates, args[0].Value)
			}
		case "Area":
			if len(args) > 0 {
				attrSpec.area = args[0].Value
			}
		}
	}
	return attrSpec
}

// parseActionAttributes returns a list of actionSpecs containing routing information derived from the action's attributes
func parseActionAttributes(method MethodDeclaration) []actionSpec {
	allAttrs := method.Attributes()

	if _, nonAction := allAttrs["Microsoft.AspNetCore.Mvc.NonAction"]; nonAction {
		return []actionSpec{}
	}

	verbAttrs := allAttrs.OfType(
		"Microsoft.AspNetCore.Mvc.HttpGet",
		"Microsoft.AspNetCore.Mvc.HttpPost",
		"Microsoft.AspNetCore.Mvc.HttpPut",
		"Microsoft.AspNetCore.Mvc.HttpPatch",
		"Microsoft.AspNetCore.Mvc.HttpDelete",
		"Microsoft.AspNetCore.Mvc.HttpHead",
		"Microsoft.AspNetCore.Mvc.HttpOptions",
	)
	routeAttrs := allAttrs.OfType(
		"Microsoft.AspNetCore.Mvc.Route",
	)

	acceptVerbsAttrs := allAttrs.OfType("Microsoft.AspNetCore.Mvc.AcceptVerbs")

	// actions without any attributes are only matched if the controller has a route and there are no other matching routes
	if len(verbAttrs) == 0 && len(routeAttrs) == 0 && len(acceptVerbsAttrs) == 0 {
		return []actionSpec{{
			name:             method.Name,
			method:           method,
			hasRouteTemplate: false,
			verb:             core.VerbAny,
		}}
	}
	var specs []actionSpec

	// handle verb attributes
	for _, verbAttr := range verbAttrs {
		_, attrName := splitQualifiedName(verbAttr.Name)
		verb := core.Verb(strings.ToUpper(strings.TrimPrefix(attrName, "Http")))
		if _, supported := core.Verbs[verb]; !supported {
			continue // unsupported verb
		}
		args := verbAttr.Args()
		routeTemplate := ""
		if len(args) > 0 && args[0].Name == "" {
			routeTemplate = args[0].Value
		}

		spec := actionSpec{
			name:             method.Name,
			method:           method,
			routeTemplate:    routeTemplate,
			verb:             verb,
			hasRouteTemplate: len(args) > 0,
		}
		specs = append(specs, spec)
	}

	allRoutesVerbs := make(map[core.Verb]struct{})

	// handle [AcceptVerbs] attributes
	for _, acceptAttr := range acceptVerbsAttrs {
		var verbs []core.Verb
		args := acceptAttr.Args()
		routeTemplate := ""
		hasRouteTemplate := false
		for _, arg := range args {
			if arg.Name == "" {
				verb := core.Verb(strings.ToUpper(arg.Value))
				_, supported := core.Verbs[verb]

				if supported {
					verbs = append(verbs, verb)
				}
			}
			if arg.Name == "Route" {
				routeTemplate = arg.Value
				hasRouteTemplate = true
			}
		}

		for _, verb := range verbs {
			if hasRouteTemplate {
				spec := actionSpec{
					verb:             verb,
					name:             method.Name,
					method:           method,
					routeTemplate:    routeTemplate,
					hasRouteTemplate: hasRouteTemplate,
				}
				specs = append(specs, spec)
			} else {
				// apply all non-route-specific verb restrictions to all [Route] attributes
				allRoutesVerbs[verb] = struct{}{}
			}
		}
	}

	if len(allRoutesVerbs) == 0 {
		// default to "ANY" if no verb restrictions exist
		allRoutesVerbs[core.VerbAny] = struct{}{}
	}

	// handle [Route] attributes
	for _, routeAttr := range routeAttrs {
		args := routeAttr.Args()
		routeTemplate := ""
		if len(args) > 0 && args[0].Name == "" {
			routeTemplate = args[0].Value
		}

		for verb := range allRoutesVerbs {
			spec := actionSpec{
				name:             method.Name,
				method:           method,
				routeTemplate:    routeTemplate,
				verb:             verb,
				hasRouteTemplate: len(args) > 0,
			}
			specs = append(specs, spec)
		}
	}

	return specs
}
