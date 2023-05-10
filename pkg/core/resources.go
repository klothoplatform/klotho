package core

import (
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/mitchellh/mapstructure"
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
		// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
		KlothoConstructRef() []AnnotationKey
		// Id returns the id of the cloud resource
		Id() ResourceId
	}

	CreateableResource[P any] interface {
		Resource
		// Create modifies itself to be a completed instance of the resource given a set of metadata
		Create(dag *ResourceGraph, params P) error
	}

	ResourceId struct {
		Provider string
		Type     string
		Name     string
	}

	// CloudResourceLink describes what Resources are necessary to ensure that a dependency between two Constructs are satisfied at an infrastructure level
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

	HasOutputFiles interface {
		GetOutputFiles() []File
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}
)

const (
	ALL_RESOURCES_IAC_VALUE = "*"
)

func (id ResourceId) String() string {
	return id.Provider + ":" + id.Type + ":" + id.Name
}

func GetMapDecoder(result interface{}) *mapstructure.Decoder {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, Result: result})
	if err != nil {
		panic(err)
	}
	return decoder
}
