package core

import (
	"strings"

	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
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

	// ExpandableResource is a resource that can generate its own dependencies. See [CreateResource].
	ExpandableResource[K any] interface {
		Resource
		Create(dag *ResourceGraph, param K) error
	}

	ResourceId struct {
		Provider string
		Type     string
		// Namespace is optional and is used to disambiguate resources that might have
		// the same name. It can also be used to associate an imported resource with
		// a specific namespace such as a subnet to a VPC.
		Namespace string
		Name      string
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

	// InternalProvider is used for resources that don't directly correspond to a deployed resource,
	// but are used to convey data or metadata about resources that should be respected during IaC rendering.
	// A notable usage is for imported resources.
	//? Do we want to revisit how to accomplish this? It was originally implemented to avoid duplicated
	// fields or methods across various resources.
	InternalProvider = "internal"
)

func (id ResourceId) String() string {
	s := id.Provider + ":" + id.Type
	if id.Namespace != "" {
		s += ":" + id.Namespace
	}
	return s + ":" + id.Name
}

func (id ResourceId) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

func (id *ResourceId) UnmarshalText(data []byte) error {
	parts := strings.Split(string(data), ":")
	if len(parts) < 3 || len(parts) > 4 {
		return errors.Errorf("invalid number of parts (%d) in resource id '%s'", len(parts), string(data))
	}
	id.Provider = parts[0]
	id.Type = parts[1]
	if len(parts) == 4 {
		id.Namespace = parts[2]
		id.Name = parts[3]
	} else {
		id.Name = parts[2]
	}
	return nil
}

func (id ResourceId) MarshalTOML() ([]byte, error) {
	return id.MarshalText()
}

func (id *ResourceId) UnmarshalTOML(data []byte) error {
	return id.UnmarshalText(data)
}

func GetMapDecoder(result interface{}) *mapstructure.Decoder {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, Result: result})
	if err != nil {
		panic(err)
	}
	return decoder
}
