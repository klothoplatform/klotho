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

		// AnnotationCapability returns the annotation capability of the construct. This helps us tie the annotation types to the constructs for the time being
		AnnotationCapability() string
	}

	// BaseConstructsRefSet is a set of BaseConstructs
	BaseConstructSet map[BaseConstruct]struct{}

	// Resource describes a resource at the provider, infrastructure level
	Resource interface {
		BaseConstruct
		// BaseConstructsRef returns a set of BaseConstructs which caused the creation of this resource
		BaseConstructsRef() BaseConstructSet
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

func IsConstructOfAnnotationCapability(baseConstruct BaseConstruct, cap string) bool {
	cons, ok := baseConstruct.(Construct)
	if !ok {
		return false
	}
	return cons.AnnotationCapability() == cap
}

func ListAllConstructs() []Construct {
	return []Construct{
		&ExecutionUnit{},
		&Gateway{},
		&StaticUnit{},
		&Orm{},
		&PubSub{},
		&Secrets{},
		&Kv{},
		&Fs{},
		&Config{},
		&RedisCluster{},
		&RedisNode{},
	}
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

func (s *BaseConstructSet) Add(k BaseConstruct) {
	if *s == nil {
		*s = make(BaseConstructSet)
	}
	(*s)[k] = struct{}{}
}

func (s BaseConstructSet) Has(k BaseConstruct) bool {
	_, ok := s[k]
	return ok
}

func (s BaseConstructSet) Delete(k BaseConstruct) {
	delete(s, k)
}

func (s *BaseConstructSet) AddAll(ks BaseConstructSet) {
	for k := range ks {
		s.Add(k)
	}
}

func (s BaseConstructSet) Clone() BaseConstructSet {
	clone := make(BaseConstructSet)
	clone.AddAll(s)
	return clone
}

func (s BaseConstructSet) CloneWith(ks BaseConstructSet) BaseConstructSet {
	clone := make(BaseConstructSet)
	clone.AddAll(s)
	clone.AddAll(ks)
	return clone
}

func BaseConstructSetOf(keys ...BaseConstruct) BaseConstructSet {
	s := make(BaseConstructSet)
	for _, k := range keys {
		s.Add(k)
	}
	return s
}
