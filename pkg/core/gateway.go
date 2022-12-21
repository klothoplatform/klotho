package core

type (
	Gateway struct {
		Name   string
		GWType string
		Routes []Route
		// Map of gateway targets with the exec unit name as the key
		Targets       map[string]GatewayTarget
		DefinedIn     string
		ExportVarName string
	}

	// TODO: have a conversation around how we want to handle cloud resources we generate on behalf of the user but want in dependency graphs
	// Intermediary resource which is not necessarily explicilty created by the user but necessary for gateway integration.
	// Example is an NLB for ECS instances that want to use expose.
	GatewayTarget struct {
		ExecUnitName string
		Kind         string
	}

	Route struct {
		// Path should be expressed using Express's route path syntax or a subset thereof
		// (see: http://expressjs.com/en/4x/api.html#path-examples)
		Path         string
		ExecUnitName string
		Verb         Verb
		// HandledInFile is the path to the file which this route is defined/handled in
		HandledInFile string
	}

	// Verb is the HTTP verb used in the route. May be upper or lower case, users
	// are expected to convert to their needs.
	Verb string
)

var (
	GatewayKind             = "gateway"
	NetworkLoadBalancerKind = "nlb"

	Verbs = map[Verb]struct{}{
		"ANY":     {},
		"GET":     {},
		"POST":    {},
		"PUT":     {},
		"DELETE":  {},
		"PATCH":   {},
		"OPTIONS": {},
		"HEAD":    {},
	}
)

func (gw *Gateway) Type() string { return gw.GWType }
func NewGateway(name string) *Gateway {
	return &Gateway{
		Name:    name,
		Targets: make(map[string]GatewayTarget),
	}
}

func (gw *Gateway) Key() ResourceKey {
	return ResourceKey{
		Name: gw.Name,
		Kind: GatewayKind,
	}
}

func (it *GatewayTarget) Key() ResourceKey {
	return ResourceKey{
		Name: it.ExecUnitName,
		Kind: it.Kind,
	}
}

func (it *GatewayTarget) Type() string { return it.Kind }

func (gw *Gateway) AddRoute(route Route, unit *ExecutionUnit, targetKind string) (string, *GatewayTarget) {
	target, ok := gw.Targets[route.ExecUnitName]
	for _, r := range gw.Routes {
		if r.Path == route.Path && r.Verb == route.Verb {
			target = gw.Targets[r.ExecUnitName]
			return r.ExecUnitName, &target
		}
	}

	if targetKind != "" && !ok {
		target = GatewayTarget{
			ExecUnitName: route.ExecUnitName,
			Kind:         targetKind,
		}
		gw.Targets[route.ExecUnitName] = target
	}

	gw.Routes = append(gw.Routes, route)
	return "", &target
}

func FindUpstreamGateways(unit *ExecutionUnit, result *CompilationResult, deps *Dependencies) []*Gateway {
	var upstreamGateways []*Gateway
	for _, dep := range deps.Upstream(unit.Key()) {
		res := result.Get(dep)
		if gw, ok := res.(*Gateway); ok {
			upstreamGateways = append(upstreamGateways, gw)
		} else if gwt, ok := res.(*GatewayTarget); ok {
			for _, gwtDep := range deps.Upstream(gwt.Key()) {
				if gw, ok := result.Get(gwtDep).(*Gateway); ok {
					upstreamGateways = append(upstreamGateways, gw)
				}
			}
		}
	}
	return upstreamGateways
}
