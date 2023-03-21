package core

import (
	"github.com/klothoplatform/klotho/pkg/graph"
)

type (
	Construct interface {
		Provenance() AnnotationKey
		Id() string
	}

	CloudResource interface {
		// Provider returns name of the provider the resource is correlated to
		Provider() string
		// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
		KlothoConstructRef() []AnnotationKey
		// ID returns the id of the cloud resource
		Id() string
	}

	CloudResourceLink interface {
		// Dependency returns the klotho resource dependencies this link correlates to
		Dependency() *graph.Edge[Construct] // Edge in the klothoconstructDag
		// Resources returns a set of resources which make up the Link
		Resources() map[CloudResource]struct{}
		// Type returns type of link, correlating to its Link ID
		Type() string
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}
)

func Get(g *graph.Directed[Construct], key AnnotationKey) Construct {
	return g.GetVertex(key.ToString())
}

func GetResourcesOfCapability(g *graph.Directed[Construct], capability string) (filtered []Construct) {
	vertices := g.GetAllVertices()
	for _, v := range vertices {
		if v.Provenance().Capability == capability {
			filtered = append(filtered, v)

		}
	}
	return
}

func GetResourcesOfType[T Construct](g *graph.Directed[Construct]) (filtered []T) {
	vertices := g.GetAllVertices()
	for _, v := range vertices {
		if vT, ok := v.(T); ok {
			filtered = append(filtered, vT)
		}
	}
	return
}

func GetExecUnitForPath(g *graph.Directed[Construct], fp string) (*ExecutionUnit, File) {
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
