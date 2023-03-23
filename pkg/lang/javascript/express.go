package javascript

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type ExpressHandler struct {
	output expressOutput
	log    *zap.Logger
	Config *config.Application
}

type expressMiddleware struct {
	UseExpr      *sitter.Node
	ObjectName   string
	PropertyName string
	Path         string
	f            *core.SourceFile
}

type expressOutput struct {
	middleware []query.Reference
	verbs      []query.Reference
	listeners  []expressListner
}

type expressListner struct {
	varName      string
	f            *core.SourceFile
	appName      string
	annotationId string
}

type routeMethodPath struct {
	Verb string
	Path string
	f    *core.SourceFile
}

func (p ExpressHandler) Name() string { return "Express" }

func (p ExpressHandler) Transform(input *core.InputFiles, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](constructGraph) {
		err := p.transformSingle(constructGraph, unit)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (p *ExpressHandler) transformSingle(constructGraph *core.ConstructGraph, unit *core.ExecutionUnit) error {

	execUnitInfo := execUnitExposeInfo{Unit: unit, RoutesByGateway: make(map[gatewaySpec][]gatewayRouteDefinition)}
	p.output = expressOutput{}
	p.log = zap.L().With(zap.String("unit", unit.ID))

	var errs multierr.Error

	for _, f := range unit.Files() {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}
		newF, err := p.handleFile(js, unit)
		if err != nil {
			errs.Append(err)
			continue
		}
		if newF != nil {
			unit.Add(newF)
		}
	}
	for _, f := range unit.Files() {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}
		p.queryResources(js)
	}
	err := p.assignRoutesToGateway(&execUnitInfo, constructGraph)
	errs.Append(err)

	handleGatewayRoutes(&execUnitInfo, constructGraph, p.log)
	return errs.ErrOrNil()
}

func (p *ExpressHandler) handleFile(f *core.SourceFile, unit *core.ExecutionUnit) (*core.SourceFile, error) {
	annots := f.Annotations()

	fileContent := string(f.Program())
	for _, annot := range annots {
		log := zap.L().With(logging.AnnotationField(annot), logging.FileField(f))
		cap := annot.Capability
		if cap.Name != annotation.ExposeCapability {
			continue
		}

		if cap.ID == "" {
			return nil, core.NewCompilerError(f, annot, errors.New("'id' is required"))
		}

		// target can be public or private for now
		// currently private is unimplemented, so
		// we fail unless it's set to public
		// TODO: we should also link to documentation when
		// it's available
		target, ok := cap.Directives.String("target")
		if !ok {
			target = "private"
		}
		if target != "public" {
			return nil, core.NewCompilerError(f, annot, errors.New("expose capability must specify target = \"public\""))
		}

		listen := findListener(annot)

		if listen.Expression == nil {
			log.Debug("No listener found")
			continue
		}

		appName, err := findApp(listen)
		if err != nil {
			return nil, core.NewCompilerError(f, annot, errors.New("Couldn't find expose app creation"))
		}

		actedOn, newfileContent := p.actOnAnnotation(f, &listen, fileContent, appName, p.Config.GetResourceType(unit), cap.ID)
		if actedOn {
			fileContent = newfileContent
			err := f.Reparse([]byte(fileContent))
			if err != nil {
				return f, errors.Wrap(err, "error reparsing after substitutions")
			}
		}
	}

	return f, nil
}

func (h *ExpressHandler) actOnAnnotation(f *core.SourceFile, listen *exposeListenResult, fileContent string, appName string, unitType string, id string) (actedOn bool, newfileContent string) {

	varName := h.findExpress(f)
	listenVarName := listen.Identifier.Content()
	actedOn = false
	newfileContent = fileContent
	if varName != listenVarName {
		return
	}
	//TODO: look into moving this runtime-specific logic elsewhere
	if unitType == "lambda" {
		newfileContent = CommentNodes(fileContent, listen.Expression.Content())
	}
	h.output.listeners = append(h.output.listeners, expressListner{varName: listenVarName, f: f, appName: appName, annotationId: id})
	newfileContent += fmt.Sprintf(`
	exports.%s = %s
	`, strings.TrimPrefix(appName, "exports."), appName)
	actedOn = true
	return
}

func (h *ExpressHandler) findExpress(f *core.SourceFile) string {
	nextMatch := DoQuery(f.Tree().RootNode(), expressApp)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		varName, expressName, exports := match["var"], match["express"], match["exports"]

		if exports != nil && !query.NodeContentEquals(exports, "exports") {
			continue
		}

		imp := FindImportForVar(f.Tree().RootNode(), expressName.Content())
		if imp.Source != `express` {
			continue
		}

		return varName.Content()
	}
	return ""
}

func (h *ExpressHandler) assignRoutesToGateway(info *execUnitExposeInfo, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	fileContentUpdate := make(map[*core.SourceFile]string)
	for _, listener := range h.output.listeners {
		f := listener.f
		listenVarName := listener.varName

		gwSpec := gatewaySpec{
			FilePath:   f.Path(),
			AppVarName: listener.appName,
			gatewayId:  listener.annotationId,
		}
		info.RoutesByGateway[gwSpec] = []gatewayRouteDefinition{}

		localRoutes := h.handleLocalRoutes(listener.f, listenVarName, "", info.Unit.ID)
		if len(localRoutes) > 0 {
			h.log.Sugar().Infof("Found %d route(s) on server '%s'", len(localRoutes), listenVarName)
			info.RoutesByGateway[gwSpec] = append(info.RoutesByGateway[gwSpec], localRoutes...)
		}

		middleware := h.findAllRouterMWs(listenVarName, f.Path())

		h.log.Sugar().Debugf("Found %d middleware", len(middleware))

		for _, mw := range middleware {
			mwImportName := listenVarName
			if mw.ObjectName != "" {
				mwImportName = mw.ObjectName
				h.log = h.log.With(logging.NodeField(mw.UseExpr))
			}
			imp := FindImportForVar(f.Tree().RootNode(), mwImportName)

			var routes []gatewayRouteDefinition
			if imp == (Import{}) {
				// not an imported router, assume it's defined locally
				routes = h.handleLocalRoutes(mw.f, mwImportName, mw.Path, info.Unit.ID)
			} else {
				path := imp.Source
				if path[0] != '.' {
					h.log.Sugar().Debugf("Skipping non-relative (%s) imported middleware", path)
					continue
				}

				mwFile := GetFileForModule(constructGraph, path)

				var exportName string
				if mw.PropertyName == "" {
					h.log.Debug("Looking for default export")
					exportNode := FindDefaultExport(mwFile.Tree().RootNode())
					if exportNode == nil {
						h.log.Sugar().Warnf("Could not find default export in '%s'", mwFile.Path())
						continue
					}
					exportName = exportNode.Content()
				} else {
					h.log.Sugar().Debugf("Looking for export of %s", mw.PropertyName)
					exportNode := FindExportForVar(
						mwFile.Tree().RootNode(),
						mw.PropertyName,
					)
					if exportNode == nil {
						h.log.Sugar().Warnf("Could not find export for '%s' in '%s'", mw.PropertyName, mwFile.Path())
						continue
					}
					exportName = exportNode.Content()
				}

				if mwUnit := core.FileExecUnitName(mwFile); mwUnit != "" && info.Unit.ID != mwUnit {
					importAssign := imp.ImportNode
					mwUse := mw.UseExpr. // call_expression
								Parent() // expression_statement
					fileContent, ok := fileContentUpdate[mw.f]
					if !ok {
						fileContent = string(mw.f.Program())
					}

					fileContent = CommentNodes(fileContent, importAssign.Content(), mwUse.Content())
					fileContentUpdate[mw.f] = fileContent
					continue // we have no routes to add, and make sure we don't log the "no routes" warning
				} else {
					// TODO check if mw is a Router and only handle the local routes if so
					routes = h.handleLocalRoutes(mwFile, exportName, mw.Path, info.Unit.ID)
				}
			}
			if len(routes) == 0 {
				h.log.Sugar().Warnf("No routes found for middleware '%s'", mwImportName)
			} else {
				h.log.Sugar().Infof("Found %d route(s) for middleware '%s'", len(routes), mwImportName)
			}
			info.RoutesByGateway[gwSpec] = append(info.RoutesByGateway[gwSpec], routes...)
		}
		for f, content := range fileContentUpdate {
			err := f.Reparse([]byte(content))
			if err != nil {
				errs.Append(err)
			}
		}
	}
	return errs.ErrOrNil()
}

func (h *ExpressHandler) handleLocalRoutes(f *core.SourceFile, varName string, routePrefix string, unitName string) (routes []gatewayRouteDefinition) {
	log := h.log.With(logging.FileField(f))

	verbFuncs := h.findVerbFuncs(varName)
	localVerbFuncs := []routeMethodPath{}
	for _, v := range verbFuncs {
		if v.f.Path() == f.Path() {
			localVerbFuncs = append(localVerbFuncs, v)
		}
	}

	log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	if len(localVerbFuncs) == 0 && routePrefix != "" {
		// TODO this could result in incorrect behaviour if varName is not a Router (ie, some other type of middleware)
		log.Sugar().Infof("Adding in catchall route for prefix '%s' for middleware '%s'", routePrefix, varName)
		routes = []gatewayRouteDefinition{
			{
				Route: core.Route{
					Path:          routePrefix,
					ExecUnitName:  unitName,
					Verb:          core.Verb("ANY"),
					HandledInFile: f.Path(),
				},
				DefinedInPath: f.Path(),
			},
			{
				Route: core.Route{
					// use a greedy parameter so all requests are routed
					// NOTE: do not use "proxy" as the parameter name which will truncate
					// the routePrefix by serverless-express
					Path:          path.Join(routePrefix, "/:rest*"),
					ExecUnitName:  unitName,
					Verb:          core.Verb("ANY"),
					HandledInFile: f.Path(),
				},
				DefinedInPath: f.Path(),
			},
		}
	}

	for _, vfunc := range localVerbFuncs {
		routePath := sanitizeExpressPath(path.Join(routePrefix, vfunc.Path))
		if routePath == "/:rest*" || routePath == ":rest*" {
			rootRoute := core.Route{
				Verb:          core.Verb(vfunc.Verb),
				Path:          "/",
				ExecUnitName:  unitName,
				HandledInFile: vfunc.f.Path(),
			}
			log.Sugar().Debugf("Found catch-all route function %s %s for '%s'", rootRoute.Verb, rootRoute.Path, varName)
			routes = append(routes, gatewayRouteDefinition{
				Route:         rootRoute,
				DefinedInPath: vfunc.f.Path(),
			})
		}
		route := core.Route{
			Verb:          core.Verb(vfunc.Verb),
			Path:          routePath,
			ExecUnitName:  unitName,
			HandledInFile: vfunc.f.Path(),
		}
		log.Sugar().Debugf("Found route function %s %s for '%s'", route.Verb, route.Path, varName)
		routes = append(routes, gatewayRouteDefinition{
			Route:         route,
			DefinedInPath: vfunc.f.Path(),
		})
	}
	return
}

func (h *ExpressHandler) queryResources(f *core.SourceFile) {
	h.output.middleware = append(h.output.middleware, query.FindReferencesInFile(
		f,
		exposeRouters,
		validateMw,
	)...)

	h.output.verbs = append(h.output.verbs, query.FindReferencesInFile(
		f,
		exposeVerb,
		validateVerbs,
	)...)
}

func (h *ExpressHandler) findAllRouterMWs(listenerName string, listnerImportPath string) []expressMiddleware {

	var mw []expressMiddleware
	for _, res := range h.output.middleware {
		f := res.File
		match := res.QueryResult

		if f.Path() == listnerImportPath {
			if !query.NodeContentEquals(match["obj"], listenerName) {
				continue
			}
		} else {
			obj := strings.Split(match["obj"].Content(), ".")
			importVar := obj[0]

			imp := FindImportForVar(f.Tree().RootNode(), importVar)

			relPath, err := filepath.Rel(filepath.Dir(f.Path()), listnerImportPath)
			if err != nil {
				continue
			}

			if FileToLocalModule(imp.Source) != FileToLocalModule(relPath) {
				continue
			}
		}

		m := expressMiddleware{
			UseExpr:    match["expr"],
			ObjectName: match["mwObj"].Content(),
			f:          f,
		}

		if mwProp := match["mwProp"]; mwProp != nil {
			m.PropertyName = mwProp.Content()
		}

		if path := match["path"]; path != nil {
			m.Path = StringLiteralContent(path)
		}

		mw = append(mw, m)
	}
	return mw
}

func (h *ExpressHandler) findVerbFuncs(varName string) []routeMethodPath {

	var mw []routeMethodPath
	for _, res := range h.output.verbs {

		file := res.File
		match := res.QueryResult

		obj, prop, path := match["obj"], match["prop"], match["path"]

		if !query.NodeContentEquals(obj, varName) {
			continue
		}

		verb := prop.Content()
		if verb == "all" {
			verb = "any"
		}

		mw = append(mw, routeMethodPath{
			Verb: verb,
			Path: StringLiteralContent(path),
			f:    file,
		})
	}
	return mw
}

// Validation methods

func validateVerbs(match map[string]*sitter.Node, f *core.SourceFile) bool {
	prop := match["prop"]
	funcName := prop.Content()
	if funcName == "all" {
		funcName = "any"
	}
	_, supported := core.Verbs[core.Verb(strings.ToUpper(funcName))]
	return supported
}

func validateMw(match map[string]*sitter.Node, f *core.SourceFile) bool {
	prop := match["prop"]

	switch {
	case !query.NodeContentEquals(prop, "use"):
		return false
	}

	return true
}

var rootWildcardRegex = regexp.MustCompile(`^/?\*$`)
var wildcardSuffixRegex = regexp.MustCompile(`/\*$`)

func sanitizeExpressPath(path string) string {

	// replace '/*' or '*' with /:rest*
	path = rootWildcardRegex.ReplaceAllString(path, "/:rest*")

	// replace '{prefix}/*' with '{prefix}/:rest*' -- {prefix} is optional
	path = wildcardSuffixRegex.ReplaceAllString(path, "/:rest*")

	return path
}
