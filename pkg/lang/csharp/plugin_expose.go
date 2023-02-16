package csharp

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
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
		FilePath        string
		AppBuilderName  string
		MapsControllers bool
		gatewayId       string
	}
	gatewayRouteDefinition struct {
		core.Route
		DefinedInPath string
	}

	aspDotNetCoreHandler struct {
		Result           *core.CompilationResult
		Deps             *core.Dependencies
		Unit             *core.ExecutionUnit
		RoutesByGateway  map[gatewaySpec][]gatewayRouteDefinition
		RootPath         string
		log              *zap.Logger
		ControllerRoutes []gatewayRouteDefinition
	}
	actionSpec struct {
		method           MethodDeclaration
		verb             core.Verb
		routeTemplate    string
		name             string
		hasRouteTemplate bool // an empty string is a valid routeTemplate value
	}

	controllerSpec struct {
		execUnitName string
		name         string
		class        TypeDeclaration
		actions      []actionSpec
		controllerAttributeSpec
	}

	controllerAttributeSpec struct {
		routeTemplates []string
		area           string
	}
)

// useEndpointsResult represents an ASP.net Core IApplicationBuilder.UseEndpoints() invocation
type useEndpointsResult struct {
	StartupClassDeclaration        *sitter.Node // Declaration of the Startup class surrounding the expose annotation
	UseExpression                  *sitter.Node // Expression of the UseEndpoints() invocation (app.UseEndpoints(endpoints => {...})
	AppBuilderIdentifier           *sitter.Node // Identifier of the builder (IApplicationBuilder app)
	EndpointRouteBuilderIdentifier *sitter.Node // Identifier of the RoutesBuilder param (endpoints => {...})
}

const (
	builderNamespace = "Microsoft.AspNetCore.Builder"
	hostingNamespace = "Microsoft.AspNetCore.Hosting"
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
	h := &aspDotNetCoreHandler{Result: result, Deps: deps, RoutesByGateway: make(map[gatewaySpec][]gatewayRouteDefinition)}
	err := h.handle(unit)
	if err != nil {
		err = core.WrapErrf(err, "ASP.NET Core handler failed for %s", unit.Name)
	}

	return err
}

func (h *aspDotNetCoreHandler) handle(unit *core.ExecutionUnit) error {
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
						ExecUnitName:  unit.Name,
						Verb:          core.VerbAny,
						HandledInFile: spec.FilePath,
					},
					DefinedInPath: spec.FilePath,
				},
				{
					Route: core.Route{
						Path:          "/:proxy*",
						ExecUnitName:  unit.Name,
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
				targetUnit = unit.Name
			}
			h.Deps.Add(gw.Key(), core.ResourceKey{Name: targetUnit, Kind: core.ExecutionUnitKind})
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
			var appBuilderName string = useEndpoint.AppBuilderIdentifier.Content()
			var endpointRouteBuilderName string = useEndpoint.EndpointRouteBuilderIdentifier.Content()

			gwSpec := gatewaySpec{
				FilePath:        f.Path(),
				AppBuilderName:  appBuilderName,
				gatewayId:       capability.ID,
				MapsControllers: isMapControllersInvoked(useEndpoint) && areControllersInjected(useEndpoint.StartupClassDeclaration),
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
			zap.L().Sugar().Debugf("Mapped route %s %s from %s", route.Verb, route.Path, c.name)
		}
	}

	return f, nil
}

func findIApplicationBuilder(cap *core.Annotation) []useEndpointsResult {
	var results []useEndpointsResult
	nextMatch := DoQuery(query.FirstAncestorOfType(cap.Node, "class_declaration"), exposeStartupConfigure)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		if !IsValidTypeName(match["param1_type"], builderNamespace, "IApplicationBuilder") ||
			!IsValidTypeName(match["param2_type"], hostingNamespace, "IWebHostEnvironment") {
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
				StartupClassDeclaration:        match["class_declaration"],
			})
		}
	}

	return results
}

func isMapControllersInvoked(useEndpoints useEndpointsResult) bool {
	_, found := DoQuery(useEndpoints.UseExpression, fmt.Sprintf(exposeMapControllersFormat, useEndpoints.EndpointRouteBuilderIdentifier.Content()))()
	return found
}

type routeMethodPath struct {
	Verb core.Verb
	Path string
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
			Path: stringLiteralContent(routePath),
		})
	}
	return route, err
}

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
		route := core.Route{
			Verb: core.Verb(vfunc.Verb),
			// using lowercase routes enforces consistency
			// since ASP.NET core is case-insensitive and API Gateway is case-sensitive
			Path:          strings.ToLower(sanitizeConventionalPath(path.Join("/", h.RootPath, prefix, vfunc.Path))),
			ExecUnitName:  h.Unit.Name,
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

func (h *aspDotNetCoreHandler) findControllersInFile(file *core.SourceFile) []controllerSpec {
	// controller docs: https://learn.microsoft.com/en-us/aspnet/core/mvc/controllers/actions?view=aspnetcore-7.0
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
			name:                    controller.Name,
			class:                   controller,
			controllerAttributeSpec: parseControllerAttributes(controller),
			actions:                 findActionsInController(controller),
			execUnitName:            h.Unit.Name,
		}
		controllerSpecs = append(controllerSpecs, spec)
	}
	return controllerSpecs
}

func (c controllerSpec) resolveRoutes() []gatewayRouteDefinition {
	shortName := strings.TrimSuffix(c.name, "Controller")
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
			// route templates starting with "~" indicate prefixes should be ignored
			if strings.HasPrefix(routeTemplate, "~") {
				routeTemplate = strings.TrimPrefix(routeTemplate, "~")
			} else {
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

			routes = append(routes, gatewayRouteDefinition{
				Route: core.Route{
					Verb: verb,
					// using lowercase routes enforces consistency
					// since ASP.NET core is case-insensitive and API Gateway is case-sensitive
					Path:          strings.ToLower(sanitizeAttributeBasedPath(routeTemplate, c.area, shortName, action.name)),
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

func parseControllerAttributes(controller TypeDeclaration) controllerAttributeSpec {
	attrs := controller.Attributes().OfType("Route", "Area")
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

	var routePrefixes []string

	for _, routeAttr := range routeAttrs {
		args := routeAttr.Args()
		routeTemplate := ""
		if len(args) > 0 && args[0].Name == "" {
			routeTemplate = args[0].Value
		}

		// when [HTTP<VERB>] or [AcceptVerbs] attributes are present [Route] attributes are treated as prefixes
		if len(verbAttrs) > 0 || len(acceptVerbsAttrs) > 0 {
			routePrefixes = append(routePrefixes, routeTemplate)
			continue
		}

		spec := actionSpec{
			name:             method.Name,
			method:           method,
			routeTemplate:    routeTemplate,
			verb:             core.VerbAny,
			hasRouteTemplate: len(args) > 0,
		}
		specs = append(specs, spec)
	}

	if len(routePrefixes) == 0 {
		routePrefixes = append(routePrefixes, "") // fall back to empty prefix
	}

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

		for _, prefix := range routePrefixes {
			spec := actionSpec{
				name:             method.Name,
				method:           method,
				routeTemplate:    path.Join(prefix, routeTemplate),
				verb:             verb,
				hasRouteTemplate: len(args) > 0,
			}
			specs = append(specs, spec)
		}
	}

	for _, acceptAttr := range acceptVerbsAttrs {
		var verbs []core.Verb
		args := acceptAttr.Args()
		routeTemplate := ""
		hasRouteTemplate := false
		for _, arg := range args {
			if arg.Name == "" {
				verb := core.Verb(strings.ToUpper(arg.Value))
				_, supported := core.Verbs[verb]

				if supported == true {
					verbs = append(verbs, verb)
				}
			}
			if arg.Name == "Route" {
				routeTemplate = arg.Value
				hasRouteTemplate = true
			}
		}

		for _, verb := range verbs {
			for _, prefix := range routePrefixes {
				spec := actionSpec{
					verb:             verb,
					name:             method.Name,
					method:           method,
					routeTemplate:    path.Join(prefix, routeTemplate),
					hasRouteTemplate: hasRouteTemplate,
				}
				specs = append(specs, spec)
			}
		}
	}
	return specs
}

// sanitizeConventionalPath converts ASP.net conventional path parameters to Express syntax,
// but does not perform validation to ensure that the supplied string is a valid ASP.net route.
// As such, there's no expectation of correct output for invalid paths
// Regexp constraints and controller/action token replacement are not yet supported
func sanitizeConventionalPath(path string) string {
	// convert to longest possible proxy route when required
	firstProxyParamIndex := findFirstProxyRouteIndicator(path)
	if firstProxyParamIndex > -1 {
		path = path[0:firstProxyParamIndex]
		path = path[0:strings.LastIndex(path, "{")+1] + "rest*}"
	}

	// convert path params to express syntax
	path = regexp.MustCompile("{([^:}]*):?[^}]*}").ReplaceAllString(path, ":$1")
	return path
}

func sanitizeAttributeBasedPath(path string, area string, controller string, action string) string {
	//TODO: handle regex constraints -- they may include additional curly braces ("{", "}") that aren't currently accounted for

	// replace params such as {controller=Index}
	specialParamFormat := `(?i){\s*%s\s*=\s*%s\s*}`
	path = regexp.MustCompile(fmt.Sprintf(specialParamFormat, "area", area)).ReplaceAllString(path, area)
	path = regexp.MustCompile(fmt.Sprintf(specialParamFormat, "controller", controller)).ReplaceAllString(path, controller)
	path = regexp.MustCompile(fmt.Sprintf(specialParamFormat, "action", action)).ReplaceAllString(path, action)

	// convert to longest possible proxy route when required
	firstProxyParamIndex := findFirstProxyRouteIndicator(path)
	if firstProxyParamIndex > -1 {
		path = path[0:firstProxyParamIndex]
		path = path[0:strings.LastIndex(path, "{")+1] + "rest*}"
	}

	// convert path params to express syntax
	path = regexp.MustCompile("{([^:}]*):?[^}]*}").ReplaceAllString(path, ":$1")

	// replace special tokens
	path = regexp.MustCompile(`(?i)\[area]`).ReplaceAllString(path, area)
	path = regexp.MustCompile(`(?i)\[controller]`).ReplaceAllString(path, controller)
	path = regexp.MustCompile(`(?i)\[action]`).ReplaceAllString(path, action)
	return path
}

func findFirstProxyRouteIndicator(path string) int {
	firstProxyParamIndex := -1
	for _, i := range []int{
		strings.Index(path, "?"),
		strings.Index(path, "="),
		strings.Index(path, "*"),
	} {
		if i >= 0 && (firstProxyParamIndex < 0 || i < firstProxyParamIndex) {
			firstProxyParamIndex = i
		}
	}
	return firstProxyParamIndex
}
