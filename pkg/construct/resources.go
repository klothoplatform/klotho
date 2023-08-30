package construct

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/mitchellh/mapstructure"
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
		// Functionality returns the functionality of the construct. This helps us determine how to expand constructs
		Functionality() Functionality
		// Attributes returns the attributes of the construct. This helps us determine how to expand constructs
		Attributes() map[string]any
	}

	BaseConstructSet map[ResourceId]BaseConstruct

	// DeleteContext is supposed to tell us when we are able to delete a resource based on its dependencies
	DeleteContext struct {
		// RequiresNoUpstream is a boolean that tells us if deletion relies on there being no upstream resources
		RequiresNoUpstream bool `yaml:"requires_no_upstream" toml:"requires_no_upstream"`
		// RequiresNoDownstream is a boolean that tells us if deletion relies on there being no downstream resources
		RequiresNoDownstream bool `yaml:"requires_no_downstream" toml:"requires_no_downstream"`
		// RequiresExplicitDelete is a boolean that tells us if deletion relies on the resource being explicitly deleted
		RequiresExplicitDelete bool `yaml:"requires_explicit_delete" toml:"requires_explicit_delete"`
		// RequiresNoUpstreamOrDownstream is a boolean that tells us if deletion relies on there being no upstream or downstream resources
		RequiresNoUpstreamOrDownstream bool `yaml:"requires_no_upstream_or_downstream" toml:"requires_no_upstream_or_downstream"`
	}

	// Resource describes a resource at the provider, infrastructure level
	Resource interface {
		BaseConstruct
		BaseConstructRefs() BaseConstructSet
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

	// IaCValue is a struct that defines a value we need to grab from a specific resource. It is up to the plugins to make the determination of how to retrieve the value
	IaCValue struct {
		// ResourceId is the resource the IaCValue is correlated to
		ResourceId ResourceId
		// Property defines the intended characteristic of the resource we want to retrieve
		Property string
	}

	HasOutputFiles interface {
		GetOutputFiles() []io.File
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}

	Functionality string

	resourceResolver interface {
		GetResource(id ResourceId) Resource
	}
)

type (
	// ResourceTemplate defines how rules are handled by the engine in terms of making sure they are functional in the graph
	ResourceTemplate struct {
		// Provider refers to the resources provider
		Provider string `json:"provider" yaml:"provider"`
		// Type refers to the unique type identifier of the resource
		Type string `json:"type" yaml:"type"`
		// Rules defines a set of rules that must pass checks and actions which must be carried out to make a resource operational
		Rules []OperationalRule `json:"rules" yaml:"rules"`
		// Configuration specifies how to act on any intrinsic values of a resource to make it operational
		Configuration []Configuration `json:"configuration" yaml:"configuration"`
		// SanitizationRules defines a set of rules that are used to ensure the resource's name is valid
		NameSanitization Sanitization `json:"sanitization" yaml:"sanitization"`
		// DeleteContext defines the context in which a resource can be deleted
		DeleteContext DeleteContext `json:"delete_context" yaml:"delete_context"`
	}

	// OperationalRule defines a rule that must pass checks and actions which must be carried out to make a resource operational
	OperationalRule struct {
		// Enforcement defines how the rule should be enforced
		Enforcement OperationEnforcement `json:"enforcement" yaml:"enforcement"`
		// Direction defines the direction of the rule. The direction options are upstream or downstream
		Direction Direction `json:"direction" yaml:"direction"`
		// ResourceTypes defines the resource types that the rule should be enforced on. Resource types must be specified if classifications is not specified
		ResourceTypes []string `json:"resource_types" yaml:"resource_types"`
		// Classifications defines the classifications that the rule should be enforced on. Classifications must be specified if resource types is not specified
		Classifications []string `json:"classifications" yaml:"classifications"`
		// SetField defines the field on the resource that should be set to the resource that satisfies the rule
		SetField string `json:"set_field" yaml:"set_field"`
		// RemoveDirectDependency defines if the direct dependency between the resource and the rule's resource(s) that satisfies the rule should be removed.
		// We also use this flag to determine if we are retrieving direct dependencies or all dependencies in the specified direction, when looking for rule satisfaction.
		RemoveDirectDependency bool `json:"remove_direct_dependency" yaml:"remove_direct_dependency"`
		// NumNeeded defines the number of resources that must satisfy the rule
		NumNeeded int `json:"num_needed" yaml:"num_needed"`
		// Rules defines a set of sub rules that will be carried out based on the evaluation of the initial parent rule
		Rules []OperationalRule `json:"rules" yaml:"rules"`
		// UnsatisfiedAction defines what action should be taken if the rule is not satisfied
		UnsatisfiedAction UnsatisfiedAction `json:"unsatisfied_action" yaml:"unsatisfied_action"`

		// NoParentDependency is a flag to signal if a sub rule is not supposed to add a dependency on the resource that satisfies the parent rule
		NoParentDependency bool `json:"no_parent_dependency" yaml:"no_parent_dependency"`
	}

	// UnsatisfiedAction defines what action should be taken if the rule is not satisfied
	UnsatisfiedAction struct {
		// Operation defines what action should be taken if the rule is not satisfied
		Operation UnsatisfiedActionOperation `json:"operation" yaml:"operation"`
		// DefaultType defines the default type of resource that should be acted upon if the rule is not satisfied
		DefaultType string `json:"default_type" yaml:"default_type"`
		// Unique defines if the resource that is created should be unique
		Unique bool `json:"unique" yaml:"unique"`
	}

	// Configuration defines how to act on any intrinsic values of a resource to make it operational
	Configuration struct {
		// Fields defines a field that should be set on the resource
		Field string `json:"field" yaml:"field"`
		// Value defines the value that should be set on the resource
		Value any `json:"value" yaml:"value"`
		// ZeroValueAllowed defines if the value can be set to the zero value of the field
		ZeroValueAllowed bool `json:"zero_value_allowed" yaml:"zero_value_allowed"`
	}

	Sanitization struct {
		Rules     []SanitizationRule `json:"rules" yaml:"rules"`
		MaxLength int                `json:"max_length" yaml:"max_length"`
		MinLength int                `json:"min_length" yaml:"min_length"`
	}
	SanitizationRule struct {
		Pattern     string `json:"pattern" yaml:"pattern"`
		Replacement string `json:"replacement" yaml:"replacement"`
	}

	// OperationEnforcement defines how the rule should be enforced
	OperationEnforcement string
	// UnsatisfiedActionOperation defines what action should be taken if the rule is not satisfied
	UnsatisfiedActionOperation string
	// Direction defines the direction of the rule. The direction options are upstream or downstream
	Direction string
)

const (
	// ExactlyOne defines that the rule should be enforced on exactly one resource
	ExactlyOne OperationEnforcement = "exactly_one"
	// Conditional defines that the rule should be enforced on a resource if it exists
	Conditional OperationEnforcement = "conditional"
	// AnyAvailable defines that the rule should be enforced on any available resource
	AnyAvailable OperationEnforcement = "any_available"

	// CreateUnsatisfiedResource defines that a resource should be created if the rule is not satisfied
	CreateUnsatisfiedResource UnsatisfiedActionOperation = "create"
	// ErrorUnsatisfiedResource defines that an error should be returned if the rule is not satisfied
	ErrorUnsatisfiedResource UnsatisfiedActionOperation = "error"

	Upstream   Direction = "upstream"
	Downstream Direction = "downstream"
)

func (or *OperationalRule) String() string {
	if or.ResourceTypes != nil {
		return fmt.Sprintf("%s %s", or.Enforcement, or.ResourceTypes)
	} else if or.Classifications != nil {
		return fmt.Sprintf("%s %s", or.Enforcement, or.Classifications)
	}
	return string(or.Enforcement)
}

const (
	Compute   Functionality = "compute"
	Cluster   Functionality = "cluster"
	Storage   Functionality = "storage"
	Api       Functionality = "api"
	Messaging Functionality = "messaging"
	Unknown   Functionality = "Unknown"

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

func GetMapDecoder(result interface{}) *mapstructure.Decoder {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{ErrorUnset: true, Result: result})
	if err != nil {
		panic(err)
	}
	return decoder
}

func (s *BaseConstructSet) Add(k BaseConstruct) {
	if k == nil {
		return
	}
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

// GetResourcesReflectively looks at a resource and determines all resources which appear as internal properties to the resource
func GetResourcesReflectively(resolver resourceResolver, source Resource) []Resource {
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
				resources = append(resources, getNestedResources(resolver, source, elemValue)...)
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				resources = append(resources, getNestedResources(resolver, source, elemValue)...)
			}

		default:
			resources = append(resources, getNestedResources(resolver, source, fieldValue)...)
		}
	}
	return resources
}

func IsResourceChild(resolver resourceResolver, source Resource, target Resource) bool {
	visited := map[Resource]bool{}
	return isResourceChild(resolver, source, target, visited)
}

func isResourceChild(resolver resourceResolver, source Resource, target Resource, visited map[Resource]bool) bool {
	visited[source] = true
	if source == target {
		return true
	}
	if source == nil {
		return false
	}
	sourceValue := reflect.ValueOf(source)
	sourceType := sourceValue.Type()
	if sourceType.Kind() == reflect.Pointer {
		sourceValue = sourceValue.Elem()
		sourceType = sourceType.Elem()
	}
	for i := 0; i < sourceType.NumField(); i++ {
		if sourceValue.Field(i).Type().Name() == "BaseConstructSet" {
			continue
		}
		fieldValue := sourceValue.Field(i)
		switch fieldValue.Kind() {
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < fieldValue.Len(); elemIdx++ {
				elemValue := fieldValue.Index(elemIdx)
				nestedResources := getNestedResources(resolver, source, elemValue)
				for _, nestedResource := range nestedResources {
					if visited[nestedResource] {
						continue
					}
					if isResourceChild(resolver, nestedResource, target, visited) {
						return true
					}
				}
			}

		case reflect.Map:
			for iter := fieldValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				nestedResources := getNestedResources(resolver, source, elemValue)
				for _, nestedResource := range nestedResources {
					if visited[nestedResource] {
						continue
					}
					if isResourceChild(resolver, nestedResource, target, visited) {
						return true
					}
				}
			}

		default:
			nestedResources := getNestedResources(resolver, source, fieldValue)
			for _, nestedResource := range nestedResources {
				if visited[nestedResource] {
					continue
				}
				if isResourceChild(resolver, nestedResource, target, visited) {
					return true
				}
			}
		}
	}
	return false
}

// getNestedResources gets all resources which exist as attributes on a BaseConstruct by using reflection
func getNestedResources(resolver resourceResolver, source BaseConstruct, targetValue reflect.Value) (resources []Resource) {
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
		if !value.ResourceId.IsZero() {
			resource := resolver.GetResource(value.ResourceId)
			if resource != nil {
				return []Resource{resource}
			}
		}
	case ResourceId:
		return []Resource{resolver.GetResource(value)}
	default:
		correspondingValue := targetValue
		for correspondingValue.Kind() == reflect.Pointer {
			correspondingValue = targetValue.Elem()
		}
		switch correspondingValue.Kind() {

		case reflect.Struct:
			for i := 0; i < correspondingValue.NumField(); i++ {
				childVal := correspondingValue.Field(i)
				resources = append(resources, getNestedResources(resolver, source, childVal)...)
			}
		case reflect.Slice, reflect.Array:
			for elemIdx := 0; elemIdx < correspondingValue.Len(); elemIdx++ {
				elemValue := correspondingValue.Index(elemIdx)
				resources = append(resources, getNestedResources(resolver, source, elemValue)...)
			}

		case reflect.Map:
			for iter := correspondingValue.MapRange(); iter.Next(); {
				elemValue := iter.Value()
				resources = append(resources, getNestedResources(resolver, source, elemValue)...)
			}

		}
	}
	return
}
