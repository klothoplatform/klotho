package javascript

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/query"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type gatewaySpec struct {
	FilePath   string
	AppVarName string
	gatewayId  string
}

type gatewayRouteDefinition struct {
	core.Route
	DefinedInPath string
}

type execUnitExposeInfo struct {
	Unit            *core.ExecutionUnit
	RoutesByGateway map[gatewaySpec][]gatewayRouteDefinition
}

// An exposeListenResult represents a javascript listen call's nodes
type exposeListenResult struct {
	Expression *sitter.Node // Expression of the listen result (app.listen(3000, () => {}))
	Identifier *sitter.Node // Identifier of the listen result (app)
}

func findListener(cap *core.Annotation) exposeListenResult {

	nextMatch := DoQuery(cap.Node, exposeListener)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		prop := match["prop"]

		if prop.Content() == "listen" {
			return exposeListenResult{
				Expression: match["expression"],
				Identifier: match["identifier"],
			}
		}
	}

	return exposeListenResult{}
}

func handleGatewayRoutes(info *execUnitExposeInfo, constructGraph *core.ConstructGraph, log *zap.Logger) {
	for spec, routes := range info.RoutesByGateway {
		gw := core.NewGateway(spec.gatewayId)
		if existing := constructGraph.GetConstruct(gw.Id()); existing != nil {
			gw = existing.(*core.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppVarName
			constructGraph.AddConstruct(gw)
		}
		if len(routes) == 0 && len(gw.Routes) == 0 {
			log.Sugar().Infof("Adding catchall route for gateway %+v with no detected routes", spec)
			routes = []gatewayRouteDefinition{
				{
					Route: core.Route{
						Path:          "/",
						ExecUnitName:  info.Unit.Name,
						Verb:          core.Verb("ANY"),
						HandledInFile: spec.FilePath,
					},
					DefinedInPath: spec.FilePath,
				},
				{
					Route: core.Route{
						Path:          "/:proxy*",
						ExecUnitName:  info.Unit.Name,
						Verb:          core.Verb("ANY"),
						HandledInFile: spec.FilePath,
					},
					DefinedInPath: spec.FilePath,
				},
			}
		}
		for _, route := range routes {
			existsInUnit := gw.AddRoute(route.Route, info.Unit)
			if existsInUnit != "" {
				log.Sugar().Infof("Not adding duplicate route %v for %v. Exists in %v", route.Path, route.ExecUnitName, existsInUnit)
				continue
			}

			targetFile := info.Unit.Get(route.DefinedInPath)
			targAST, ok := Language.ID.CastFile(targetFile)
			if !ok {
				continue
			}
			targetUnit := core.FileExecUnitName(targAST)
			if targetUnit == "" {
				// if the target file is in all units, direct the API gateway to use the unit that defines the listener
				targetUnit = info.Unit.Name
			}
			depKey := core.ResourceId{
				Provider: core.AbstractConstructProvider,
				Type:     annotation.ExecutionUnitCapability,
				Name:     targetUnit,
			}
			if constructGraph.GetConstruct(depKey) == nil {
				// The unit defined in the target does not exist, fall back to current one (for running in single-exec mode).
				// TODO when there are ways to combine units, we'll need a more sophisticated way to see which unit the target maps to.
				depKey.Name = info.Unit.Name
			}
			constructGraph.AddDependency(gw.Id(), depKey)
		}
	}
}

// findApp finds the variable containing the listen call for the purpose of adding the export statement
func findApp(listener exposeListenResult) (name string, err error) {
	listenName := listener.Identifier.Content()

	if listener.Expression.Parent().Type() == "program" {
		// app is use top-level, and is not a promise
		name = listenName
		return
	} else if afunc := query.FirstAncestorOfType(listener.Expression, "arrow_function"); afunc != nil {
		for n := afunc; n != nil; n = n.Parent() {
			if n.Type() != "call_expression" {
				continue
			}
			fn := n.ChildByFieldName("function")
			if fn == nil {
				continue
			}
			prop := fn.ChildByFieldName("property")
			if prop.Content() != "then" {
				continue
			}
			obj := fn.ChildByFieldName("object")
			name = obj.Content()

			return
		}
		err = errors.Errorf("unable to find variable name from arrow_function for '%s'", listenName)
	} else if fdecl := query.FirstAncestorOfType(listener.Expression, "function_declaration"); fdecl != nil {
		program := listener.Expression
		for program.Parent() != nil {
			program = program.Parent()
		}

		funcName := fdecl.ChildByFieldName("name").Content()

		next := DoQuery(program, `(variable_declarator
			name: (identifier) @name
			value: (call_expression
				function: (identifier) @func
			)
		)`)
		for {
			match, found := next()
			if !found {
				err = errors.Errorf("no variable declarators for listen in function %s", funcName)
				return
			}
			if match["func"].Content() != funcName {
				continue
			}
			name = match["name"].Content()

			break
		}
	} else {
		err = errors.Errorf("unable to determine how to find variable name for '%s'", listenName)
	}

	return
}
