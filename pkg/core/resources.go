package core

import (
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type (
	// BaseConstruct is an abstract concept for some node-type-thing in a resource-ish graph. More concretely, it is
	// either a Construct or a Resource.
	BaseConstruct interface {
		// Id returns the unique id of the construct
		Id() ResourceId
	}

	// Construct describes a resource at the source code, Klotho annotation level
	Construct interface {
		BaseConstruct
		// Provenance returns the AnnotationKey that the construct was created by
		Provenance() AnnotationKey
	}

	// Resource describes a resource at the provider, infrastructure level
	Resource interface {
		BaseConstruct
		// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
		KlothoConstructRef() AnnotationKeySet
	}

	// ExpandableResource is a resource that can generate its own dependencies. See [CreateResource].
	ExpandableResource[K any] interface {
		Resource
		Create(dag *ResourceGraph, params K) error
	}

	ConfigurableResource[K any] interface {
		Resource
		Configure(params K) error
	}

	ResourceId struct {
		Provider string `yaml:"provider" toml:"provider"`
		Type     string `yaml:"type" toml:"type"`
		// Namespace is optional and is used to disambiguate resources that might have
		// the same name. It can also be used to associate an imported resource with
		// a specific namespace such as a subnet to a VPC.
		Namespace string `yaml:"namespace" toml:"namespace"`
		Name      string `yaml:"name" toml:"name"`
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

	// ConstructId is an AnnotationKey that can turn itself into a ResourceId. It is provided as a convenience type.
	// If you have an AnnotationId and want a ResourceId corresponding to that annotation's abstract construct, do:
	//
	// 	resourceId := ConstructId(consId).ToRid()
	ConstructId AnnotationKey
)

const (
	ALL_RESOURCES_IAC_VALUE = "*"

	// InternalProvider is used for resources that don't directly correspond to a deployed resource,
	// but are used to convey data or metadata about resources that should be respected during IaC rendering.
	// A notable usage is for imported resources.
	//? Do we want to revisit how to accomplish this? It was originally implemented to avoid duplicated
	// fields or methods across various resources.
	InternalProvider = "internal"

	// AbstractConstructProvider is the provider for abstract constructs — those that don't correspond to deployable
	// resources directly, but instead expand into other constructs.
	AbstractConstructProvider = "klotho"
)

func (cid ConstructId) ToRid() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     cid.Capability,
		Name:     cid.ID,
	}
}

func IsConstructOfCapability(baseConstruct BaseConstruct, cap string) bool {
	cons, ok := baseConstruct.(Construct)
	if !ok {
		return false
	}
	return cons.Provenance().Capability == cap
}

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
