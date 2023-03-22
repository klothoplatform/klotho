package core

import (
	"github.com/klothoplatform/klotho/pkg/graph"
)

type (
	// Construct describes a resource at the source code, Klotho annotation level
	Construct interface {
		// Provenance returns the AnnotationKey that the construct was created by
		Provenance() AnnotationKey
		// Id returns the unique Id of the construct
		Id() string
	}

	// Resource describes a resource at the provider, infrastructure level
	Resource interface {
		// Provider returns name of the provider the resource is correlated to
		Provider() string
		// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
		KlothoConstructRef() []AnnotationKey
		// ID returns the id of the cloud resource
		Id() string
	}

	// CloudResourceLink describes what Resources are necessary to ensure that a depoendency between two Constructs are satisfied at an infrastructure level
	CloudResourceLink interface {
		// Dependency returns the klotho resource dependencies this link correlates to
		Dependency() *graph.Edge[Construct] // Edge in the klothoconstructDag
		// Resources returns a set of resources which make up the Link
		Resources() map[Resource]struct{}
		// Type returns type of link, correlating to its Link ID
		Type() string
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}
	ConstructGraph = graph.Directed[Construct]
	ResourceGraph  = graph.Directed[Resource]
)

func Get(g *ConstructGraph, key AnnotationKey) Construct {
	return g.GetVertex(key.ToString())
}

func GetResourcesOfCapability(g *ConstructGraph, capability string) (filtered []Construct) {
	vertices := g.GetAllVertices()
	for _, v := range vertices {
		if v.Provenance().Capability == capability {
			filtered = append(filtered, v)

		}
	}
	return
}

func GetResourcesOfType[T Construct](g *ConstructGraph) (filtered []T) {
	vertices := g.GetAllVertices()
	for _, v := range vertices {
		if vT, ok := v.(T); ok {
			filtered = append(filtered, vT)
		}
	}
	return
}

func GetExecUnitForPath(g *ConstructGraph, fp string) (*ExecutionUnit, File) {
	var best *ExecutionUnit
	var bestFile File
	for _, eu := range GetResourcesOfType[*ExecutionUnit](g) {
		f := eu.Get(fp)
		if f != nil {
			astF, ok := f.(*SourceFile)
			if ok && (best == nil || FileExecUnitName(astF) == eu.Provenance().ID) {
				best = eu
				bestFile = f
			}
		}
	}
	return best, bestFile
}

func FindUpstreamGateways(unit *ExecutionUnit, g *ConstructGraph) []*Gateway {
	gateways := []*Gateway{}
	vertices := g.IncomingVertices(unit)
	for _, v := range vertices {
		gw, ok := v.(*Gateway)
		if ok {
			gateways = append(gateways, gw)
		}
	}
	return gateways
}
