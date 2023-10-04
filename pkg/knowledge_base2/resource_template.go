package knowledgebase2

import (
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"gopkg.in/yaml.v3"
)

type (
	// ResourceTemplate defines how rules are handled by the engine in terms of making sure they are functional in the graph
	ResourceTemplate struct {
		QualifiedTypeName string `json:"qualified_type_name" yaml:"qualified_type_name"`

		Properties Properties `json:"properties" yaml:"properties"`

		Classification Classification `json:"classification" yaml:"classification"`

		// DeleteContext defines the context in which a resource can be deleted
		DeleteContext DeleteContext `json:"delete_context" yaml:"delete_context"`
		// Views defines the views that the resource should be added to as a distinct node
		Views map[string]string `json:"views" yaml:"views"`
	}

	Properties map[string]Property

	Property struct {
		Name string `json:"name" yaml:"name"`
		// Type defines the type of the property
		Type string `json:"type" yaml:"type"`

		Namespace bool `json:"namespace" yaml:"namespace"`

		DefaultValue any `json:"default_value" yaml:"default_value"`

		Required bool `json:"required" yaml:"required"`

		ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`

		DeployTime bool `json:"deploy_time" yaml:"deploy_time"`

		OperationalRule *OperationalRule `json:"operational_rule" yaml:"operational_rule"`

		Properties map[string]Property `json:"properties" yaml:"properties"`

		Path string `json:"-" yaml:"-"`
	}

	Classification struct {
		Is    []string `json:"is"`
		Gives []Gives  `json:"gives"`
	}

	Gives struct {
		Attribute     string
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
	Compute   Functionality = "compute"
	Cluster   Functionality = "cluster"
	Storage   Functionality = "storage"
	Api       Functionality = "api"
	Messaging Functionality = "messaging"
	Unknown   Functionality = "Unknown"
)

func (p *Properties) UnmarshalYAML(n *yaml.Node) error {
	type h Properties
	var p2 h
	err := n.Decode(&p2)
	if err != nil {
		return err
	}
	for name, property := range p2 {
		property.Name = name
		property.Path = name
		setChildPaths(&property, name)
		p2[name] = property
	}
	*p = Properties(p2)
	return nil
}

func setChildPaths(property *Property, currPath string) {
	for name, child := range property.Properties {
		child.Name = name
		path := currPath + "." + name
		child.Path = path
		setChildPaths(&child, path)
		property.Properties[name] = child
	}
}

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

func (template ResourceTemplate) Id() construct.ResourceId {
	args := strings.Split(template.QualifiedTypeName, ":")
	return construct.ResourceId{
		Provider: args[0],
		Type:     args[1],
	}
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

func (template ResourceTemplate) GetNamespacedProperty() *Property {
	for _, property := range template.Properties {
		if property.Namespace {
			return &property
		}
	}
	return nil
}

func (template ResourceTemplate) GetProperty(name string) *Property {
	fields := strings.Split(name, ".")
	properties := template.Properties
	for i, field := range fields {
		currFieldName := strings.Split(field, "[")[0]
		found := false
		for _, property := range properties {
			if property.Name != currFieldName {
				continue
			}
			found = true
			if len(fields) == i+1 {
				return &property
			} else {
				pType, err := property.PropertyType()
				if err != nil {
					return nil
				}
				// If the property types are a list or map, without sub fields
				//  we want to just return the property since we are setting an index or key of the end value
				switch p := pType.(type) {
				case *MapPropertyType:
					if p.Value != "" {
						return &Property{
							Type: p.Value,
							Path: name,
						}
					}
				case *ListPropertyType:
					if p.Value != "" {
						return &Property{
							Type: p.Value,
							Path: name,
						}
					}
				}
				properties = property.Properties
			}
		}
		if !found {
			return nil
		}
	}
	return nil
}
