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

	// IaCValue is a struct that defines a value we need to grab from a specific resource. It is up to the plugins to make the determination of how to retrieve the value
	IaCValue struct {
		// Resource is the resource the IaCValue is correlated to
		Resource Resource
		// Property defines the intended characteristic of the resource we want to retrieve
		Property string
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}
)
