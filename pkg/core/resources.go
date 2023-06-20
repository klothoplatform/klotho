package core

import (
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
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

	BaseConstructSet map[ResourceId]BaseConstruct

	// DeleteContext is supposed to tell us when we are able to delete a resource based on its dependencies
	DeleteContext struct {
		// RequiresNoUpstream is a boolean that tells us if deletion relies on there being no upstream resources
		RequiresNoUpstream bool
		// RequiresNoDownstream is a boolean that tells us if deletion relies on there being no downstream resources
		RequiresNoDownstream bool
		// RequiresExplicitDelete is a boolean that tells us if deletion relies on the resource being explicitly deleted
		RequiresExplicitDelete bool
		// RequiresNoUpstreamOrDownstream is a boolean that tells us if deletion relies on there being no upstream or downstream resources
		RequiresNoUpstreamOrDownstream bool
	}

	// Resource describes a resource at the provider, infrastructure level
	Resource interface {
		BaseConstruct
		BaseConstructsRef() BaseConstructSet
		DeleteContext() DeleteContext
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
	IaCValue interface {
		// Resource is the resource the IaCValue is correlated to
		Resource() Resource
		// Property defines the intended characteristic of the resource we want to retrieve
		Property() string
		// SetResource sets the resource the IaCValue is correlated to
		SetResource(Resource)
		yaml.Marshaler
		yaml.Unmarshaler
	}

	HasOutputFiles interface {
		GetOutputFiles() []File
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}

	Capability string
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
	(*s)[k.Id()] = k
}

func (s BaseConstructSet) Has(k ResourceId) bool {
	_, ok := s[k]
	return ok
}

func (s BaseConstructSet) Delete(k BaseConstruct) {
	delete(s, k.Id())
}

func (s *BaseConstructSet) AddAll(ks BaseConstructSet) {
	for _, c := range ks {
		s.Add(c)
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

func GetResourcesReflectively(source Resource) []Resource {
	resources := []Resource{}
	sourceValue := reflect.ValueOf(source)
	sourceType := sourceValue.Type()
	if sourceType.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
		sourceType = sourceType.Elem()
	}
	for i := 0; i < sourceType.NumField(); i++ {
		fieldValue := sourceValue.Field(i)
		switch fieldValue.Kind() {
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < fieldValue.Len(); elemIdx++ {
				elemValue := fieldValue.Index(elemIdx)
				resources = append(resources, loadNestedResourcesFromIds(source, elemValue)...)
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				resources = append(resources, loadNestedResourcesFromIds(source, elemValue)...)
			}

		default:
			resources = append(resources, loadNestedResourcesFromIds(source, fieldValue)...)
		}
	}
	return resources
}

func loadNestedResourcesFromIds(source BaseConstruct, targetValue reflect.Value) (resources []Resource) {
	if targetValue.Kind() == reflect.Pointer && targetValue.IsNil() {
		return
	}
	if !targetValue.CanInterface() {
		return
	}
	switch value := targetValue.Interface().(type) {
	case Resource:
		return []Resource{value}
	case IaCValue:
		if value.Resource() != nil {
			return []Resource{value.Resource()}
		}
	default:
		correspondingValue := targetValue
		for correspondingValue.Kind() == reflect.Pointer {
			correspondingValue = targetValue.Elem()
		}
		switch correspondingValue.Kind() {

		case reflect.Struct:
			for i := 0; i < correspondingValue.NumField(); i++ {
				childVal := correspondingValue.Field(i)
				resources = append(resources, loadNestedResourcesFromIds(source, childVal)...)
			}
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < correspondingValue.Len(); elemIdx++ {
				elemValue := correspondingValue.Index(elemIdx)
				resources = append(resources, loadNestedResourcesFromIds(source, elemValue)...)
			}

		case reflect.Map:
			for iter := correspondingValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				resources = append(resources, loadNestedResourcesFromIds(source, elemValue)...)
			}

		}
	}
	return
}
