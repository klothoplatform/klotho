package core

import "github.com/dominikbraun/graph"

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
