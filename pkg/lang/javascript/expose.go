package javascript

import (
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

func findListener(cap *core.Annotation, source []byte) exposeListenResult {

	nextMatch := DoQuery(cap.Node, exposeListener)
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		prop := match["prop"]

		if prop.Content(source) == "listen" {
			return exposeListenResult{
				Expression: match["expression"],
				Identifier: match["identifier"],
			}
		}
	}

	return exposeListenResult{}
}

func handleGatewayRoutes(info *execUnitExposeInfo, result *core.CompilationResult, deps *core.Dependencies, log *zap.Logger) {
	for spec, routes := range info.RoutesByGateway {
		gw := core.NewGateway(spec.gatewayId)
		if existing := result.Get(gw.Key()); existing != nil {
			gw = existing.(*core.Gateway)
		} else {
			gw.DefinedIn = spec.FilePath
			gw.ExportVarName = spec.AppVarName
			result.Add(gw)
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
			// determine if a target is needed and which
			targetKind := ""
			switch info.Unit.Type() {
			// TODO: move these out of expose into runtime somehow
			case "fargate":
				targetKind = core.NetworkLoadBalancerKind
			}

			existsInUnit, it := gw.AddRoute(route.Route, info.Unit, targetKind)
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
			if it.Kind == "" {
				depKey := core.ResourceKey{Name: targetUnit, Kind: core.ExecutionUnitKind}
				if result.Get(depKey) == nil {
					// The unit defined in the target does not exist, fall back to current one (for running in single-exec mode).
					// TODO when there are ways to combine units, we'll need a more sophisticated way to see which unit the target maps to.
					depKey.Name = info.Unit.Name
				}
				deps.Add(gw.Key(), depKey)
			} else {
				// If an integration target exists for an exec unit, create the cloud resource and set the deps as gw -> it -> route exec unit
				if existing := result.Get(it.Key()); existing == nil {
					result.Add(it)
				}
				deps.Add(gw.Key(), it.Key())
				deps.Add(it.Key(), core.ResourceKey{Name: targetUnit, Kind: core.ExecutionUnitKind})
			}
		}
	}
}

// findApp finds the variable containing the listen call for the purpose of adding the export statement
func findApp(source []byte, listener exposeListenResult) (name string, err error) {
	listenName := listener.Identifier.Content(source)

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
			if prop.Content(source) != "then" {
				continue
			}
			obj := fn.ChildByFieldName("object")
			name = obj.Content(source)

			return
		}
		err = errors.Errorf("unable to find variable name from arrow_function for '%s'", listenName)
	} else if fdecl := query.FirstAncestorOfType(listener.Expression, "function_declaration"); fdecl != nil {
		program := listener.Expression
		for program.Parent() != nil {
			program = program.Parent()
		}

		funcName := fdecl.ChildByFieldName("name").Content(source)

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
			if match["func"].Content(source) != funcName {
				continue
			}
			name = match["name"].Content(source)

			break
		}
	} else {
		err = errors.Errorf("unable to determine how to find variable name for '%s'", listenName)
	}

	return
}
