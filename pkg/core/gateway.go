package core

type (
	Gateway struct {
		AnnotationKey
		Routes []Route
		// Map of gateway targets with the exec unit name as the key
		DefinedIn     string
		ExportVarName string
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

func (v Verb) String() string {
	return string(v)
}

const (
	VerbAny     = Verb("ANY")
	VerbGet     = Verb("GET")
	VerbPost    = Verb("POST")
	VerbPut     = Verb("PUT")
	VerbDelete  = Verb("DELETE")
	VerbPatch   = Verb("PATCH")
	VerbOptions = Verb("OPTIONS")
	VerbHead    = Verb("HEAD")
)

var (
	GatewayKind             = "gateway"
	NetworkLoadBalancerKind = "nlb"

	Verbs = map[Verb]struct{}{
		VerbAny:     {},
		VerbGet:     {},
		VerbPost:    {},
		VerbPut:     {},
		VerbDelete:  {},
		VerbPatch:   {},
		VerbOptions: {},
		VerbHead:    {},
	}
)

func NewGateway(key AnnotationKey) *Gateway {
	return &Gateway{
		AnnotationKey: key,
	}
}

func (p *Gateway) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Gateway) Id() string {
	return p.AnnotationKey.ToId()
}

func (gw *Gateway) AddRoute(route Route, unit *ExecutionUnit) string {
	for _, r := range gw.Routes {
		if r.Path == route.Path && r.Verb == route.Verb {
			return r.ExecUnitName
		}
	}

	gw.Routes = append(gw.Routes, route)
	return ""
}
