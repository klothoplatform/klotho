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
		FilePath       string
		AppBuilderName string
		gatewayId      string
	}
	gatewayRouteDefinition struct {
		core.Route
		DefinedInPath string
	}

	aspDotNetCoreHandler struct {
		Result          *core.CompilationResult
		Deps            *core.Dependencies
		Unit            *core.ExecutionUnit
		RoutesByGateway map[gatewaySpec][]gatewayRouteDefinition
		RootPath        string
		log             *zap.Logger
	}
)

// An exposeUseEndpointsResult represents an ASP.net Core IApplicationBuilder.UseEndpoints() invocation
type exposeUseEndpointsResult struct {
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
		err = core.WrapErrf(err, "express handler failure for %s", unit.Name)
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
			var appBuilderName string = useEndpoint.EndpointRouteBuilderIdentifier.Content()
			var endpointRouteBuilderName string = useEndpoint.EndpointRouteBuilderIdentifier.Content()

			gwSpec := gatewaySpec{
				FilePath:       f.Path(),
				AppBuilderName: appBuilderName,
				gatewayId:      capability.ID,
			}

			log := h.log.With(
				zap.String("IApplicationBuilder", appBuilderName),
				zap.String("IEndpointRouteBuilder", endpointRouteBuilderName),
			)

			localRoutes, err := h.findLocallyMappedRoutes(f, endpointRouteBuilderName, "")
			if err != nil {
				return nil, core.NewCompilerError(f, capAnnotation, err)
			}

			if len(localRoutes) > 0 {
				h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], localRoutes...)
			}

			if isMapControllersInvoked(useEndpoint) && areControllersInjected(useEndpoint.StartupClassDeclaration) {
				for _, csFile := range h.Unit.FilesOfLang(CSharp) {
					controllers := h.findControllersInFile(csFile)
					for _, c := range controllers {
						routes := c.resolveRoutes()
						for _, route := range c.resolveRoutes() {
							zap.L().Sugar().Debugf("Found route function %s %s for %s", route.Verb, route.Path, c.name)
						}
						h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], routes...)

					}
				}

			}
			log.Sugar().Infof("Found %d route(s) on app '%s'", len(h.RoutesByGateway[gwSpec]), appBuilderName)
		}
	}
	return f, nil
}

func findIApplicationBuilder(cap *core.Annotation) []exposeUseEndpointsResult {
	var results []exposeUseEndpointsResult
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

			results = append(results, exposeUseEndpointsResult{
				UseExpression:                  expressionMatch["expression"],
				AppBuilderIdentifier:           paramNameN,
				EndpointRouteBuilderIdentifier: expressionMatch["endpoints_param"],
				StartupClassDeclaration:        match["class_declaration"],
			})
		}
	}

	return results
}

func isMapControllersInvoked(useEndpoints exposeUseEndpointsResult) bool {
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
			Verb:          core.Verb(vfunc.Verb),
			Path:          sanitizeConventionalPath(path.Join(h.RootPath, prefix, vfunc.Path)),
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

// sanitizeConventionalPath converts ASP.net conventional path parameters to Express syntax,
// but does not perform validation to ensure that the supplied string is a valid ASP.net route.
// As such, there's no expectation of correct output for invalid paths
func sanitizeConventionalPath(path string) string {
	firstOptionalIndex := strings.Index(path, "?")
	firstDefaultIndex := strings.Index(path, "=")
	firstProxyParamIndex := firstOptionalIndex
	if firstProxyParamIndex == -1 || (firstDefaultIndex > -1 && firstDefaultIndex < firstProxyParamIndex) {
		firstProxyParamIndex = firstDefaultIndex
	}
	if firstProxyParamIndex > -1 {
		// convert to longest possible proxy route
		path = path[0:firstProxyParamIndex]
		path = path[0:strings.LastIndex(path, "{")+1] + "rest*}"
	}

	// convert path params to express syntax
	path = regexp.MustCompile("{([^:}]*):?[^}]*}").ReplaceAllString(path, ":$1")
	return path
}

func sanitizeControllerPath(path string, area string, controller string, action string) string {
	//TODO: handle regex constraints -- they may include additional curly braces ("{", "}") that aren't currently accounted for
	firstOptionalIndex := strings.Index(path, "?")
	firstDefaultIndex := strings.Index(path, "=")
	firstProxyParamIndex := firstOptionalIndex
	if firstProxyParamIndex == -1 || (firstDefaultIndex > -1 && firstDefaultIndex < firstProxyParamIndex) {
		firstProxyParamIndex = firstDefaultIndex
	}
	if firstProxyParamIndex > -1 {
		// convert to longest possible proxy route
		path = path[0:firstProxyParamIndex]
		path = path[0:strings.LastIndex(path, "{")+1] + "rest*}"
	}

	// convert path params to express syntax
	path = regexp.MustCompile("{([^:}]*):?[^}]*}").ReplaceAllString(path, ":$1")

	path = strings.ReplaceAll(path, "[area]", fmt.Sprintf(":%s", area))
	path = strings.ReplaceAll(path, "[controller]", fmt.Sprintf(":%s", controller))
	path = strings.ReplaceAll(path, "[action]", fmt.Sprintf(":%s", action))
	return path
}

type actionSpec struct {
	method        MethodDeclaration
	verb          core.Verb
	routeTemplate string
	name          string
}

type controllerSpec struct {
	execUnitName string
	name         string
	class        TypeDeclaration
	actions      []actionSpec
	controllerAttributeSpec
}

type controllerAttributeSpec struct {
	routeTemplates []string
	area           string
}

func (h *aspDotNetCoreHandler) findControllersInFile(file *core.SourceFile) []controllerSpec {
	types := FindDeclarationsInFile[*TypeDeclaration](file).Declarations()
	usingDirectives := FindImportsInFile(file)
	controllers := filter.NewSimpleFilter(HasBase("Microsoft.AspNetCore.Mvc", "ControllerBase", usingDirectives)).Apply(types...)
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
	for _, action := range c.actions {
		for _, prefix := range c.routeTemplates {
			routeTemplate := action.routeTemplate
			// route templates starting with "~" indicate prefixes should be ignored
			if strings.HasPrefix(routeTemplate, "~") {
				routeTemplate = strings.TrimPrefix(routeTemplate, "~")
			} else {
				routeTemplate = path.Join(prefix, action.routeTemplate)
			}

			routes = append(routes, gatewayRouteDefinition{
				Route: core.Route{Verb: action.verb,
					Path:          sanitizeControllerPath(routeTemplate, c.area, shortName, action.name),
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

func findActionsInController(controller TypeDeclaration) []actionSpec {
	var actions []actionSpec
	methods := FindDeclarationsAtNode[*MethodDeclaration](controller.Node).Declarations()
	for _, m := range methods {
		actions = append(actions, parseActionAttributes(*m)...)
	}
	return actions
}

func HasBase(namespace, typeName string, using Imports) predicate.Predicate[*TypeDeclaration] {
	qualifiedName := namespace + "." + typeName
	return func(d *TypeDeclaration) bool {
		if _, ok := d.Bases[qualifiedName]; ok {
			return true
		}
		for _, baseNode := range d.Bases {
			if IsValidTypeName(baseNode, namespace, typeName) {
				return true
			}
		}
		return false
	}
}

func funcName(d *TypeDeclaration, using Imports) bool {
	_, hasQualifiedCB := d.Bases["Microsoft.AspNetCore.Mvc.ControllerBase"]
	if hasQualifiedCB {
		return true
	}

	_, hasCB := d.Bases["ControllerBase"]
	validNamespaces := ContainingNamespaces(d.Node)
	usingNamespace := false
	if hasCB && !hasQualifiedCB {
		nsImports := using["Microsoft.AspNetCore.Mvc"]
		for _, i := range nsImports {
			if _, isInValidNamespace := validNamespaces[i.Namespace]; i.Namespace == "" || isInValidNamespace {
				usingNamespace = true
				break
			}
		}
	}
	hasAliasedCB := false
	if !hasCB && !hasQualifiedCB {
		if cbImports, ok := using["Microsoft.AspNetCore.Mvc.ControllerBase"]; ok {
			for _, i := range cbImports {
				if _, isInValidNamespace := validNamespaces[i.Namespace]; i.Namespace == "" || isInValidNamespace {
					if _, usesAlias := d.Bases[i.Alias]; usesAlias {
						hasAliasedCB = i.Type == ImportTypeUsingAlias
						break
					}
				}
			}
		}
	}
	return d.Kind == DeclarationKindClass && ((hasCB && usingNamespace) || hasQualifiedCB || hasAliasedCB)
}

func parseControllerAttributes(controller TypeDeclaration) controllerAttributeSpec {
	matches := AllMatches(controller.AttributesList, query.Join(exposeRouteAttribute, exposeAreaAttribute))
	attrSpec := controllerAttributeSpec{}
	for _, match := range matches {
		attr := match["attr"]
		switch attr.Content() {
		case "Route":
			attrSpec.routeTemplates = append(attrSpec.routeTemplates, stringLiteralContent(match["template"]))
		case "Area":
			attrSpec.area = stringLiteralContent(match["areaName"])
		}
	}
	return attrSpec
}

func parseActionAttributes(method MethodDeclaration) []actionSpec {
	matches := AllMatches(method.Node, query.Join(exposeRouteAttribute, httpMethodAttribute))
	var specs []actionSpec

	if len(matches) == 0 {
		verb := resolveVerbFromNamePrefix(method.Name)
		if verb != "" {
			routeTemplate := method.Name[len(verb):]
			spec := actionSpec{
				name:          routeTemplate,
				method:        method,
				routeTemplate: routeTemplate,
				verb:          verb,
			}
			specs = append(specs, spec)
		}
		return specs
	}

	hasVerbAttribute := false
	var routePrefixes []string
	for _, match := range matches {
		attrName := match["attr"].Content()
		if attrName == "Route" {
			routePrefixes = append(routePrefixes, stringLiteralContent(match["template"]))
		} else {
			hasVerbAttribute = true
		}
	}
	if len(routePrefixes) == 0 {
		routePrefixes = append(routePrefixes, "") // fall back to empty prefix
	}

	for _, match := range matches {
		attrName := match["attr"].Content()
		if attrName == "Route" && hasVerbAttribute {
			continue
		}
		if !hasVerbAttribute {
			verb := resolveVerbFromNamePrefix(method.Name)
			if verb != "" {
				routeTemplate := stringLiteralContent(match["template"])
				spec := actionSpec{
					name:          method.Name[len(verb):],
					method:        method,
					routeTemplate: routeTemplate,
					verb:          verb,
				}
				specs = append(specs, spec)
			}
			continue
		}

		//TODO: add support for 'AcceptVerbs' attribute
		verb := core.Verb("")
		if strings.HasPrefix(attrName, "Http") {
			verb = core.Verb(strings.ToUpper(strings.TrimPrefix(attrName, "Http")))
			if _, supported := core.Verbs[core.Verb(verb)]; !supported {
				continue // unsupported verb
			}
		} else {
			verb = resolveVerbFromNamePrefix(method.Name)
		}

		for _, prefix := range routePrefixes {
			routeTemplate := path.Join(prefix, stringLiteralContent(match["template"]))

			spec := actionSpec{
				name:          method.Name[len(verb):],
				method:        method,
				routeTemplate: routeTemplate,
				verb:          verb,
			}
			specs = append(specs, spec)
		}
	}
	return specs
}

func resolveVerbFromNamePrefix(name string) core.Verb {
	name = strings.ToUpper(name)
	if strings.HasPrefix(name, core.VerbGet.String()) {
		return core.VerbGet
	}
	if strings.HasPrefix(name, core.VerbPost.String()) {
		return core.VerbPost
	}
	if strings.HasPrefix(name, core.VerbPut.String()) {
		return core.VerbPut
	}
	if strings.HasPrefix(name, core.VerbPatch.String()) {
		return core.VerbPatch
	}
	if strings.HasPrefix(name, core.VerbDelete.String()) {
		return core.VerbDelete
	}
	if strings.HasPrefix(name, core.VerbHead.String()) {
		return core.VerbHead
	}
	if strings.HasPrefix(name, core.VerbOptions.String()) {
		return core.VerbOptions
	}
	return ""
}
