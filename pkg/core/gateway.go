package core

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
)

type (
	Gateway struct {
		Name   string
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

	GATEWAY_TYPE = "expose"
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

func NewGateway(name string) *Gateway {
	return &Gateway{
		Name: name,
	}
}

func (p *Gateway) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     GATEWAY_TYPE,
		Name:     p.Name,
	}
}
func (p *Gateway) AnnotationCapability() string {
	return annotation.ExposeCapability
}
func (p *Gateway) Functionality() Functionality {
	return Api
}

func (p *Gateway) Attributes() map[string]any {
	return map[string]any{}
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
