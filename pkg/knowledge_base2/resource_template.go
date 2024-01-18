package knowledgebase2

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/set"
	"gopkg.in/yaml.v3"
)

//go:generate 	mockgen -source=./resource_template.go --destination=./resource_template_mock_test.go --package=knowledgebase2

type (
	// ResourceTemplate defines how rules are handled by the engine in terms of making sure they are functional in the graph
	ResourceTemplate struct {
		// QualifiedTypeName is the qualified type name of the resource
		QualifiedTypeName string `json:"qualified_type_name" yaml:"qualified_type_name"`

		// DisplayName is the common name that refers to the resource
		DisplayName string `json:"display_name" yaml:"display_name"`

		// Properties defines the properties that the resource has
		Properties Properties `json:"properties" yaml:"properties"`

		AdditionalRules []AdditionalRule `json:"additional_rules" yaml:"additional_rules"`

		// Classification defines the classification of the resource
		Classification Classification `json:"classification" yaml:"classification"`

		// PathSatisfaction defines what paths must exist for the resource must be connected to and from
		PathSatisfaction PathSatisfaction `json:"path_satisfaction" yaml:"path_satisfaction"`

		// Consumption defines properties the resource may emit or consume from other resources it is connected to or expanded from
		Consumption Consumption `json:"consumption" yaml:"consumption"`

		// DeleteContext defines the context in which a resource can be deleted
		DeleteContext DeleteContext `json:"delete_context" yaml:"delete_context"`
		// Views defines the views that the resource should be added to as a distinct node
		Views map[string]string `json:"views" yaml:"views"`

		// NoIac defines if the resource should be ignored by the IaC engine
		NoIac bool `json:"no_iac" yaml:"no_iac"`

		// SanitizeNameTmpl defines a template that is used to sanitize the name of the resource
		SanitizeNameTmpl *SanitizeTmpl `yaml:"sanitize_name"`
	}

	// PropertyDetails defines the common details of a property
	PropertyDetails struct {
		Name string `json:"name" yaml:"name"`
		// DefaultValue has to be any because it may be a template and it may be a value of the correct type
		Namespace bool `yaml:"namespace"`
		// Required defines if the property is required
		Required bool `json:"required" yaml:"required"`
		// ConfigurationDisabled defines if the property is allowed to be configured by the user
		ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`
		// DeployTime defines if the property is only available at deploy time
		DeployTime bool `json:"deploy_time" yaml:"deploy_time"`
		// OperationalRule defines a rule that is executed at runtime to determine the value of the property
		OperationalRule *PropertyRule `json:"operational_rule" yaml:"operational_rule"`
		// Description is a description of the property. This is not used in the engine solving,
		// but is metadata returned by the `ListResourceTypes` CLI command.
		Description string `json:"description" yaml:"description"`
		// Path is the path to the property in the resource
		Path string `json:"-" yaml:"-"`

		// IsImportant is a flag to denote what properties are subjectively important to show in the InfracopilotUI via
		// the `ListResourceTypes` CLI command. This field is not & should not be used in the engine.
		IsImportant bool
	}

	// Property is an interface used to define a property that exists on a resource
	// Properties are used to define the structure of a resource and how it is configured
	// Each property implementation refers to a specific type of property, such as a string or a list, etc
	Property interface {
		// SetProperty sets the value of the property on the resource
		SetProperty(resource *construct.Resource, value any) error
		// AppendProperty appends the value to the property on the resource
		AppendProperty(resource *construct.Resource, value any) error
		// RemoveProperty removes the value from the property on the resource
		RemoveProperty(resource *construct.Resource, value any) error
		// Details returns the property details for the property
		Details() *PropertyDetails
		// Clone returns a clone of the property
		Clone() Property
		// Type returns the string representation of the type of the property, as it should appear in the resource template
		Type() string
		// GetDefaultValue returns the default value for the property,
		// pertaining to the specific data being passed in for execution
		GetDefaultValue(ctx DynamicContext, data DynamicValueData) (any, error)
		// Validate ensures the value is valid for the property to `Set` (not `Append` for collection types)
		// and returns an error if it is not
		Validate(resource *construct.Resource, value any, ctx DynamicContext) error
		// SubProperties returns the sub properties of the property, if any.
		// This is used for properties that are complex structures, such as lists, sets, or maps
		SubProperties() Properties
		// Parse parses a given value to ensure it is the correct type for the property.
		// If the given value cannot be converted to the respective property type an error is returned.
		// The returned value will always be the correct type for the property
		Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error)
		// ZeroValue returns the zero value for the property type
		ZeroValue() any
		// Contains returns true if the value contains the given value
		Contains(value any, contains any) bool
	}

	// MapProperty is an interface for properties that implement map structures
	MapProperty interface {
		// Key returns the property representing the keys of the map
		Key() Property
		// Value returns the property representing the values of the map
		Value() Property
	}

	// CollectionProperty is an interface for properties that implement collection structures
	CollectionProperty interface {
		// Item returns the structure of the items within the collection
		Item() Property
	}

	// Properties is a map of properties
	Properties map[string]Property

	// Classification defines the classification of a resource
	Classification struct {
		// Is defines the classifications that the resource belongs to
		Is []string `json:"is" yaml:"is"`
		// Gives defines the attributes that the resource gives to other resources
		Gives []Gives `json:"gives" yaml:"gives"`
	}

	// Gives defines an attribute that can be provided to other functionalities for the resource it belongs to
	Gives struct {
		// Attribute is the attribute that is given
		Attribute string
		// Functionality is the list of functionalities that the attribute is given to
		Functionality []string
	}

	// DeleteContext is supposed to tell us when we are able to delete a resource based on its dependencies
	DeleteContext struct {
		// RequiresNoUpstream is a boolean that tells us if deletion relies on there being no upstream resources
		RequiresNoUpstream bool `yaml:"requires_no_upstream" toml:"requires_no_upstream"`
		// RequiresNoDownstream is a boolean that tells us if deletion relies on there being no downstream resources
		RequiresNoDownstream bool `yaml:"requires_no_downstream" toml:"requires_no_downstream"`
		// RequiresNoUpstreamOrDownstream is a boolean that tells us if deletion relies on there being no upstream or downstream resources
		RequiresNoUpstreamOrDownstream bool `yaml:"requires_no_upstream_or_downstream" toml:"requires_no_upstream_or_downstream"`
	}

	Functionality string
)

const (
	ErrRequiredProperty = "required property %s is not set on resource %s"

	Compute   Functionality = "compute"
	Cluster   Functionality = "cluster"
	Storage   Functionality = "storage"
	Api       Functionality = "api"
	Messaging Functionality = "messaging"
	Unknown   Functionality = "Unknown"
)

func (g *Gives) UnmarshalJSON(content []byte) error {
	givesString := string(content)
	if givesString == "" {
		return nil
	}
	gives := strings.Split(givesString, ":")
	g.Attribute = strings.ReplaceAll(gives[0], "\"", "")
	if len(gives) == 1 {
		g.Functionality = []string{"*"}
		return nil
	}
	g.Functionality = strings.Split(strings.ReplaceAll(gives[1], "\"", ""), ",")
	return nil
}

func (g *Gives) UnmarshalYAML(n *yaml.Node) error {
	givesString := n.Value
	if givesString == "" {
		return nil
	}
	gives := strings.Split(givesString, ":")
	g.Attribute = strings.ReplaceAll(gives[0], "\"", "")
	if len(gives) == 1 {
		g.Functionality = []string{"*"}
		return nil
	}
	g.Functionality = strings.Split(strings.ReplaceAll(gives[1], "\"", ""), ",")
	return nil
}

func (p *Properties) Clone() Properties {
	newProps := make(Properties, len(*p))
	for k, v := range *p {
		newProps[k] = v.Clone()
	}
	return newProps
}

func (template ResourceTemplate) Id() construct.ResourceId {
	args := strings.Split(template.QualifiedTypeName, ":")
	return construct.ResourceId{
		Provider: args[0],
		Type:     args[1],
	}
}

// CreateResource creates an empty resource for the given ID, running any sanitization rules on the ID.
// NOTE: Because of sanitization, once created callers must use the resulting ID for all future operations
// and not the input ID.
func CreateResource(kb TemplateKB, id construct.ResourceId) (*construct.Resource, error) {
	rt, err := kb.GetResourceTemplate(id)
	if err != nil {
		return nil, fmt.Errorf("could not create resource: get template err: %w", err)
	}
	id.Name, err = rt.SanitizeName(id.Name)
	if err != nil {
		return nil, fmt.Errorf("could not create resource: %w", err)
	}
	return &construct.Resource{
		ID:         id,
		Properties: make(construct.Properties),
	}, nil
}

func (rt ResourceTemplate) SanitizeName(name string) (string, error) {
	if rt.SanitizeNameTmpl == nil {
		return name, nil
	}
	return rt.SanitizeNameTmpl.Execute(name)
}

func (template ResourceTemplate) GivesAttributeForFunctionality(attribute string, functionality Functionality) bool {
	for _, give := range template.Classification.Gives {
		if give.Attribute == attribute && (collectionutil.Contains(give.Functionality, string(functionality)) || collectionutil.Contains(give.Functionality, "*")) {
			return true
		}
	}
	return false
}

func (template ResourceTemplate) GetFunctionality() Functionality {
	if len(template.Classification.Is) == 0 {
		return Unknown
	}
	var functionality Functionality
	for _, c := range template.Classification.Is {
		matched := true
		alreadySet := functionality != ""
		switch c {
		case "compute":
			functionality = Compute
		case "cluster":
			functionality = Cluster
		case "storage":
			functionality = Storage
		case "api":
			functionality = Api
		case "messaging":
			functionality = Messaging
		default:
			matched = false
		}
		if matched && alreadySet {
			return Unknown
		}
	}
	if functionality == "" {
		return Unknown
	}
	return functionality
}

func (template ResourceTemplate) ResourceContainsClassifications(needs []string) bool {
	for _, need := range needs {
		if !collectionutil.Contains(template.Classification.Is, need) && template.QualifiedTypeName != need {
			return false
		}
	}
	return true
}

func (template ResourceTemplate) GetNamespacedProperty() Property {
	for _, property := range template.Properties {
		if property.Details().Namespace {
			return property
		}
	}
	return nil
}

func (template ResourceTemplate) GetProperty(path string) Property {
	fields := strings.Split(path, ".")
	properties := template.Properties
FIELDS:
	for i, field := range fields {
		currFieldName := strings.Split(field, "[")[0]
		found := false
		for name, property := range properties {
			if name != currFieldName {
				continue
			}
			found = true
			if len(fields) == i+1 {
				// use a clone resource so we can modify the name in case anywhere in the path
				// has index strings or map keys
				clone := property.Clone()
				details := clone.Details()
				details.Path = path
				return clone
			} else {
				properties = property.SubProperties()
				if len(properties) == 0 {
					if mp, ok := property.(MapProperty); ok {
						clone := mp.Value().Clone()
						details := clone.Details()
						details.Path = path
						return clone
					} else if cp, ok := property.(CollectionProperty); ok {
						clone := cp.Item().Clone()
						details := clone.Details()
						details.Path = path
						return clone
					}
				}
			}
			continue FIELDS
		}
		if !found {
			return nil
		}
	}
	return nil
}

var ErrStopWalk = errors.New("stop walk")

// ReplacePath runs a simple [strings.ReplaceAll] on the path of the property and all of its sub properties.
// NOTE: this mutates the property, so make sure to [Property.Clone] it first if you don't want that.
func ReplacePath(p Property, original, replacement string) {
	p.Details().Path = strings.ReplaceAll(p.Details().Path, original, replacement)
	for _, prop := range p.SubProperties() {
		ReplacePath(prop, original, replacement)
	}
}

func (tmpl ResourceTemplate) LoopProperties(res *construct.Resource, addProp func(Property) error) error {
	queue := []Properties{tmpl.Properties}
	var props Properties
	var errs error
	for len(queue) > 0 {
		props, queue = queue[0], queue[1:]

		propKeys := make([]string, 0, len(props))
		for k := range props {
			propKeys = append(propKeys, k)
		}
		sort.Strings(propKeys)

		for _, key := range propKeys {
			prop := props[key]
			err := addProp(prop)
			if err != nil {
				if errors.Is(err, ErrStopWalk) {
					return nil
				}
				errs = errors.Join(errs, err)
				continue
			}

			if strings.HasPrefix(prop.Type(), "list") || strings.HasPrefix(prop.Type(), "set") {
				p, err := res.GetProperty(prop.Details().Path)
				if err != nil || p == nil {
					continue
				}
				// Because lists/sets will start as empty, do not recurse into their sub-properties if its not set.
				// To allow for defaults within list objects and operational rules to be run, we will look in the property
				// to see if there are values.
				if strings.HasPrefix(prop.Type(), "list") {
					length := reflect.ValueOf(p).Len()
					for i := 0; i < length; i++ {
						subProperties := make(Properties)
						for subK, subProp := range prop.SubProperties() {
							propTemplate := subProp.Clone()
							ReplacePath(propTemplate, prop.Details().Path, fmt.Sprintf("%s[%d]", prop.Details().Path, i))
							subProperties[subK] = propTemplate
						}
						if len(subProperties) > 0 {
							queue = append(queue, subProperties)
						}
					}
				} else if strings.HasPrefix(prop.Type(), "set") {
					hs, ok := p.(set.HashedSet[string, any])
					if !ok {
						errs = errors.Join(errs, fmt.Errorf("could not cast property to set"))
						continue
					}
					for k := range hs.ToMap() {
						subProperties := make(Properties)
						for subK, subProp := range prop.SubProperties() {
							propTemplate := subProp.Clone()
							ReplacePath(propTemplate, prop.Details().Path, fmt.Sprintf("%s[%s]", prop.Details().Path, k))
							subProperties[subK] = propTemplate
						}
						if len(subProperties) > 0 {
							queue = append(queue, subProperties)
						}
					}

				}
			} else if prop.SubProperties() != nil {
				queue = append(queue, prop.SubProperties())
			}
		}
	}
	return errs
}
