package constructs

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

type (
	ConstructTemplate struct {
		Id          ConstructTemplateId         `yaml:"id"`
		Version     string                      `yaml:"version"`
		Description string                      `yaml:"description"`
		Resources   map[string]ResourceTemplate `yaml:"resources"`
		Edges       []EdgeTemplate              `yaml:"edges"`
		Inputs      map[string]InputTemplate    `yaml:"inputs"`
		Outputs     map[string]OutputTemplate   `yaml:"outputs"`
		InputRules  []InputRuleTemplate         `yaml:"input_rules"`
	}

	ConstructTemplateId struct {
		Package string `yaml:"package"`
		Name    string `yaml:"name"`
	}

	ResourceTemplate struct {
		Type       string         `yaml:"type"`
		Name       string         `yaml:"name"`
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
	}

	OutputTemplate struct {
		Name        string `yaml:"name"`
		Type        string `yaml:"type"`
		Description string `yaml:"description"`
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
