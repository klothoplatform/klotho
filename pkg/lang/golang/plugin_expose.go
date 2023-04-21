package golang

import (
	"fmt"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type (
	Expose struct {
		Config  *config.Application
		runtime Runtime
	}

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
		ConstructGraph  *core.ConstructGraph
		Unit            *core.ExecutionUnit
		RoutesByGateway map[gatewaySpec][]gatewayRouteDefinition
		RootPath        string
		log             *zap.Logger
		runtime         Runtime
	}

	chiRouterDefResult struct {
		Declaration *sitter.Node
		Identifier  *sitter.Node
		RootPath    string
	}

	HttpListener struct {
		Identifier *sitter.Node
		Expression *sitter.Node
		Address    *sitter.Node
	}

	routeMethodPath struct {
		Verb string
		Path string
	}

	routerMount struct {
		Path     string
		PkgAlias string
		PkgName  string
		FuncName string
	}
)

func (p *Expose) Name() string { return "Expose" }

func (p Expose) Transform(input *core.InputFiles, fileDeps *core.FileDependencies, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](constructGraph) {
		err := p.transformSingle(constructGraph, unit)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}

func (p *Expose) transformSingle(constructGraph *core.ConstructGraph, unit *core.ExecutionUnit) error {
	h := &restAPIHandler{ConstructGraph: constructGraph, RoutesByGateway: make(map[gatewaySpec][]gatewayRouteDefinition), runtime: p.runtime}
	err := h.handle(unit)
	if err != nil {
		err = core.WrapErrf(err, "Chi handler failure for %s", unit.ID)
	}
	return err
}

func (h *restAPIHandler) handle(unit *core.ExecutionUnit) error {
	h.Unit = unit
	h.log = zap.L().With(zap.String("unit", unit.ID))

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
		gw := core.NewGateway(core.AnnotationKey{ID: gwName, Capability: annotation.ExposeCapability})
		if existing := h.ConstructGraph.GetConstruct(gw.Id()); existing != nil {
			gw = existing.(*core.Gateway)
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
			targetFile, ok := goLang.CastFile(targetFileR)
			if !ok {
				continue
			}

			targetUnit := core.FileExecUnitName(targetFile)
			if targetUnit == "" {
				// if the target file is in all units, direct the API gateway to use the unit that defines the listener
				targetUnit = unit.ID
			}
			h.ConstructGraph.AddDependency(gw.Id(), core.AnnotationKey{ID: targetUnit, Capability: annotation.ExecutionUnitCapability}.ToId())
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
		routerName := listener.Identifier.Content()

		err = h.runtime.ActOnExposeListener(h.Unit, f, &listener, routerName)
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}

		err = h.removeNetHttpImport(f)
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}

		router, err := h.findChiRouterDefinition(f, routerName)
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
			AppVarName: routerName,
			Id:         cap.ID,
		}

		log = log.With(zap.String("var", routerName))

		localRoutes, err := h.findChiRoutesForVar(f, routerName, "")
		if err != nil {
			return nil, core.NewCompilerError(f, capNode, err)
		}

		if len(localRoutes) > 0 {
			log.Sugar().Infof("Found %d route(s) on app '%s'", len(localRoutes), routerName)
			h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], localRoutes...)
		}

		// For external routes, we work back from the mount() call to get the package being called. Then
		// we find the function which defines the extra routes within the specified package

		routerMounts := h.findChiRouterMounts(f, routerName)
		for _, m := range routerMounts {
			err := h.findChiRouterMountPackage(f, &m)
			if err != nil {
				return nil, core.NewCompilerError(f, capNode, err)
			}
			filesForPackage := FindFilesForPackageName(h.Unit, m.PkgName)
			if len(filesForPackage) == 0 {
				return nil, core.NewCompilerError(f, capNode, errors.Errorf("No files found for package [%s]", m.PkgName))
			}
			file, funcNode := h.findFileForFunctionName(filesForPackage, m.FuncName)
			if file == nil {
				return nil, core.NewCompilerError(f, capNode, errors.Errorf("No file found with function named [%s]", m.FuncName))
			}
			mountedRoutes := h.findChiRoutesInFunction(file, funcNode, m)
			if len(mountedRoutes) > 0 {
				log.Sugar().Infof("Found %d route(s) on mounted router '%s.%s'", len(mountedRoutes), m.PkgAlias, m.FuncName)
				h.RoutesByGateway[gwSpec] = append(h.RoutesByGateway[gwSpec], mountedRoutes...)
			}
		}
	}
	return f, nil
}

func (h *restAPIHandler) removeNetHttpImport(f *core.SourceFile) error {
	h.log.Info(fmt.Sprintf("searching for http imports in %s", f.Path()))
	netHttpImportName := "http"
	netHttpImport := GetNamedImportInFile(f, netHttpImportName)
	if netHttpImport.Alias != "" {
		netHttpImportName = netHttpImport.Alias
	}

	nextMatch := doQuery(f.Tree().RootNode(), fmt.Sprintf("[((package_identifier)@id (#match? @id \"%s\")) ((identifier)@id (#match? @id \"%s\"))]", netHttpImportName, netHttpImportName))

	httpUsed := false
	for {
		_, found := nextMatch()
		if !found {
			break
		}
		if found {
			httpUsed = true
			break
		}
	}

	if !httpUsed {
		err := UpdateImportsInFile(f, []Import{}, []Import{{Package: "net/http"}})
		if err != nil {
			return errors.Wrap(err, "error updating imports")
		}
	}
	return nil
}

func (h *restAPIHandler) findChiRouterDefinition(f *core.SourceFile, appName string) (chiRouterDefResult, error) {
	nextMatch := doQuery(f.Tree().RootNode(), findRouterAssignment)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		identifier, definition, declaration := match["identifier"], match["definition"], match["declaration"]

		if definition.Content() == "chi.NewRouter()" {
			foundName := identifier.Content()
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

func (h *restAPIHandler) findHttpListenAndServe(cap *core.Annotation, f *core.SourceFile) (HttpListener, error) {
	nextMatch := doQuery(cap.Node, findHttpListen)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		listenExp, addr, router, expression := match["sel_exp"], match["addr"], match["router"], match["expression"]

		if listenExp.Content() == "http.ListenAndServe" {
			return HttpListener{
				Identifier: router,
				Expression: expression,
				Address:    addr,
			}, nil
		} else {
			return HttpListener{}, errors.Errorf("Expected http.ListenAndServe but found %s", listenExp.Content())
		}
	}

	return HttpListener{}, nil
}

func (h *restAPIHandler) findChiRoutesForVar(f *core.SourceFile, varName string, prefix string) ([]gatewayRouteDefinition, error) {
	var routes = make([]gatewayRouteDefinition, 0)
	log := h.log.With(logging.FileField(f))

	verbFuncs, err := h.findVerbFuncs(f.Tree().RootNode(), varName)
	if err != nil {
		return routes, err
	}

	log.Sugar().Debugf("Got %d verb functions for '%s'", len(verbFuncs), varName)

	for _, vfunc := range verbFuncs {
		route := core.Route{
			Verb:          core.Verb(vfunc.Verb),
			Path:          path.Join(h.RootPath, prefix, vfunc.Path), //TODO: Handle Chi router path parameters conversion to express for pulumi logic
			ExecUnitName:  h.Unit.ID,
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

func (h *restAPIHandler) findVerbFuncs(root *sitter.Node, varName string) ([]routeMethodPath, error) {
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

		if !query.NodeContentEquals(appName, varName) {
			continue // wrong var (not the Chi router we're looking for)
		}

		funcName := verb.Content()

		if _, supported := core.Verbs[core.Verb(strings.ToUpper(funcName))]; !supported {
			continue // unsupported verb
		}

		pathContent := stringLiteralContent(routePath)

		route = append(route, routeMethodPath{
			Verb: verb.Content(),
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

func (h *restAPIHandler) findChiRouterMounts(f *core.SourceFile, routerName string) []routerMount {
	nextMatch := doQuery(f.Tree().RootNode(), findRouterMounts)
	var mounts = make([]routerMount, 0)

	for {
		match, found := nextMatch()
		if !found {
			break
		}

		router_name, mount, path, package_name, package_func := match["router_name"], match["mount"], match["path"], match["package_name"], match["package_func"]

		if !query.NodeContentEquals(router_name, routerName) ||
			!query.NodeContentEquals(mount, "Mount") {
			continue
		}

		mounts = append(mounts, routerMount{
			Path:     stringLiteralContent(path),
			PkgAlias: package_name.Content(),
			FuncName: package_func.Content(),
		})
	}

	return mounts
}

func (h *restAPIHandler) findChiRouterMountPackage(f *core.SourceFile, mount *routerMount) error {
	nextMatch := doQuery(f.Tree().RootNode(), findImports)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		package_id, package_path := match["package_id"], match["package_path"]

		if package_path == nil {
			continue
		}

		p := strings.Split(stringLiteralContent(package_path), "/")
		package_name := p[len(p)-1]
		if package_id != nil {
			if !query.NodeContentEquals(package_id, mount.PkgAlias) {
				continue
			}
			mount.PkgName = package_name
			return nil
		}

		if package_name == mount.PkgAlias {
			mount.PkgName = package_name
			return nil
		}

	}

	return errors.Errorf("No import package found with name or alias [%s]", mount.PkgAlias)
}

func (h *restAPIHandler) findFileForFunctionName(files []*core.SourceFile, funcName string) (f *core.SourceFile, functionNode *sitter.Node) {
	for _, f := range files {
		nextMatch := doQuery(f.Tree().RootNode(), findFunction)
		for {
			match, found := nextMatch()
			if !found {
				break
			}
			function_name, function := match["function_name"], match["function"]

			if query.NodeContentEquals(function_name, funcName) {
				return f, function
			}
		}
	}
	return
}

func (h *restAPIHandler) findChiRoutesInFunction(f *core.SourceFile, funcNode *sitter.Node, m routerMount) []gatewayRouteDefinition {
	var gatewayRoutes = make([]gatewayRouteDefinition, 0)
	log := h.log.With(logging.FileField(f))

	// This is very similar in logic to how we find the local router and verbs. The difference is for external routers, we are starting from
	// the node of the specified function and don't care about what the router name is so long as the router methods are declared within this function node
	nextMatch := doQuery(funcNode, findExposeVerb)
	var routes []routeMethodPath
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		verb := match["verb"]
		routePath := match["path"]

		funcName := verb.Content()

		if _, supported := core.Verbs[core.Verb(strings.ToUpper(funcName))]; !supported {
			continue // unsupported verb
		}

		pathContent := stringLiteralContent(routePath)

		routes = append(routes, routeMethodPath{
			Verb: verb.Content(),
			Path: pathContent,
		})
	}
	log.Sugar().Debugf("Found %d verb functions from '%s.%s'", len(routes), m.PkgAlias, m.FuncName)

	for _, vfunc := range routes {
		route := core.Route{
			Verb:          core.Verb(vfunc.Verb),
			Path:          path.Join(h.RootPath, m.Path, vfunc.Path), //TODO: Handle Chi router path parameters conversion to express for pulumi logic
			ExecUnitName:  h.Unit.ID,
			HandledInFile: f.Path(),
		}
		log.Sugar().Debugf("Found route function %s %s from '%s.%s'", route.Verb, route.Path, m.PkgAlias, m.FuncName)
		gatewayRoutes = append(gatewayRoutes, gatewayRouteDefinition{
			Route:         route,
			DefinedInPath: f.Path(),
		})
	}

	return gatewayRoutes
}
