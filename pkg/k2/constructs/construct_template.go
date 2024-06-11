package constructs

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"strings"

	"gopkg.in/yaml.v3"
)

type (
	ConstructTemplate struct {
		Id            ConstructTemplateId         `yaml:"id"`
		Version       string                      `yaml:"version"`
		Description   string                      `yaml:"description"`
		Resources     map[string]ResourceTemplate `yaml:"resources"`
		Edges         []EdgeTemplate              `yaml:"edges"`
		Inputs        map[string]InputTemplate    `yaml:"inputs"`
		Outputs       map[string]OutputTemplate   `yaml:"outputs"`
		InputRules    []InputRuleTemplate         `yaml:"input_rules"`
		resourceOrder []string
	}

	ConstructTemplateId struct {
		Package string `yaml:"package"`
		Name    string `yaml:"name"`
	}

	ResourceTemplate struct {
		Type       string         `yaml:"type"`
		Name       string         `yaml:"name"`
		Namespace  string         `yaml:"namespace"`
		Properties map[string]any `yaml:"properties"`
	}

	EdgeTemplate struct {
		From ResourceRef    `yaml:"from"`
		To   ResourceRef    `yaml:"to"`
		Data map[string]any `yaml:"data"`
	}

	InputTemplate struct {
		Name        string             `yaml:"name"`
		Type        string             `yaml:"type"`
		Description string             `yaml:"description"`
		Default     any                `yaml:"default"`
		Secret      bool               `yaml:"secret"`
		PulumiKey   string             `yaml:"pulumi_key"`
		Validation  ValidationTemplate `yaml:"validation"`
		//resourcesNode *yaml.Node
		//edgesNode     *yaml.Node
	}

	OutputTemplate struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Value       any    `yaml:"value"`
	}

	InputRuleTemplate struct {
		If   string                        `yaml:"if"`
		Then ConditionalExpressionTemplate `yaml:"then"`
		Else ConditionalExpressionTemplate `yaml:"else"`
	}

	ConditionalExpressionTemplate struct {
		Resources map[string]ResourceTemplate `yaml:"resources"`
		Edges     []EdgeTemplate              `yaml:"edges"`
		Outputs   map[string]OutputTemplate   `yaml:"outputs"`
	}

	ValidationTemplate struct {
		Required     bool     `yaml:"required"`
		MinLength    int      `yaml:"min_length"`
		MaxLength    int      `yaml:"max_length"`
		MinValue     int      `yaml:"min_value"`
		MaxValue     int      `yaml:"max_value"`
		Pattern      string   `yaml:"pattern"`
		Enum         []string `yaml:"enum"`
		UniqueValues bool     `yaml:"unique_values"`
	}
)

func (c *ConstructTemplateId) UnmarshalYAML(value *yaml.Node) error {
	// Split the value into parts
	parts := strings.Split(value.Value, ".")

	// Check if there are at least two parts: package and name
	if len(parts) < 2 {
		return fmt.Errorf("invalid construct template id: %s", value.Value)
	}

	// The name is the last part
	c.Name = parts[len(parts)-1]

	// The package is all the parts except the last one, joined by a dot
	c.Package = strings.Join(parts[:len(parts)-1], ".")

	return nil
}

func ParseConstructTemplateId(id string) (ConstructTemplateId, error) {
	// Parse a construct template id from a string
	parts := strings.Split(id, ".")
	if len(parts) < 2 {
		return ConstructTemplateId{}, fmt.Errorf("invalid construct template id: %s", id)
	}
	return ConstructTemplateId{
		Package: strings.Join(parts[:len(parts)-1], "."),
		Name:    parts[len(parts)-1],
	}, nil
}

func (c *ConstructTemplateId) String() string {
	return fmt.Sprintf("%s.%s", c.Package, c.Name)
}

func (c *ConstructTemplateId) FromURN(urn model.URN) error {
	if urn.Type != "construct" {
		return fmt.Errorf("invalid urn type: %s", urn.Type)
	}

	parts := strings.Split(urn.Subtype, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid construct template id: %s", urn.Subtype)
	}

	c.Package = strings.Join(parts[:len(parts)-1], ".")
	c.Name = parts[len(parts)-1]
	return nil
}

func (e *EdgeTemplate) UnmarshalYAML(value *yaml.Node) error {
	// Unmarshal the edge template from a yaml node
	var edge struct {
		From string         `yaml:"from"`
		To   string         `yaml:"to"`
		Data map[string]any `yaml:"data"`
	}
	if err := value.Decode(&edge); err != nil {
		return err
	}

	if interpolationPattern.MatchString(edge.From) {
		e.From = ResourceRef{
			ResourceKey: edge.From,
			Type:        ResourceRefTypeInterpolated,
		}
	} else {
		e.From = ResourceRef{
			ResourceKey: edge.From,
			Type:        ResourceRefTypeTemplate,
		}
	}

	if interpolationPattern.MatchString(edge.To) {
		e.To = ResourceRef{
			ResourceKey: edge.To,
			Type:        ResourceRefTypeInterpolated,
		}
	} else {
		e.To = ResourceRef{
			ResourceKey: edge.To,
			Type:        ResourceRefTypeTemplate,
		}

	}

	e.Data = edge.Data
	return nil
}

func (c *ConstructTemplate) UnmarshalYAML(value *yaml.Node) error {
	// alias
	type constructTemplate ConstructTemplate

	// Unmarshal the construct template from a yaml node
	var template constructTemplate
	if err := value.Decode(&template); err != nil {
		return err
	}

	var resourceOrder []string
	// Capture the resource order
	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		if keyNode.Value == "resources" {
			for j := 0; j < len(value.Content[i+1].Content); j += 2 {
				resourceOrder = append(resourceOrder, value.Content[i+1].Content[j].Value)
			}
		}
	}

	template.resourceOrder = resourceOrder
	// Convert the alias to the actual type
	*c = ConstructTemplate(template)
	return nil
}

type ResourceIterator struct {
	template *ConstructTemplate
	index    int
}

func (r *ResourceIterator) Next() (string, ResourceTemplate, bool) {
	if r.index >= len(r.template.resourceOrder) || r.index >= len(r.template.Resources) {
		return "", ResourceTemplate{}, false
	}

	// Get the next resource that actually exists in the template
	for _, ok := r.template.Resources[r.template.resourceOrder[r.index]]; !ok && r.index < len(r.template.resourceOrder); r.index++ {
		// do nothing
	}
	key := r.template.resourceOrder[r.index]
	resource := r.template.Resources[key]

	r.index++
	return key, resource, true
}

func (c *ConstructTemplate) ResourcesIterator() *ResourceIterator {
	return &ResourceIterator{
		template: c,
	}
}

func (r *ResourceIterator) ForEach(f func(string, ResourceTemplate)) {
	for key, resource, ok := r.Next(); ok; key, resource, ok = r.Next() {
		f(key, resource)
	}
}

func (r *ResourceIterator) Reset() {
	r.index = 0
}
