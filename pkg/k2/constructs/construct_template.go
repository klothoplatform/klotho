package constructs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"

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
		From ResourceRef        `yaml:"from"`
		To   ResourceRef        `yaml:"to"`
		Data construct.EdgeData `yaml:"data"`
	}

	InputTemplate struct {
		Name          string             `yaml:"name"`
		Type          string             `yaml:"type"`
		Description   string             `yaml:"description"`
		Default       any                `yaml:"default"`
		Secret        bool               `yaml:"secret"`
		PulumiKey     string             `yaml:"pulumi_key"`
		Validation    ValidationTemplate `yaml:"validation"`
		Configuration map[string]any     `yaml:"configuration"`
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

	BindingTemplate struct {
		From          ConstructTemplateId         `yaml:"from"`
		To            ConstructTemplateId         `yaml:"to"`
		Priority      int                         `yaml:"priority"`
		Inputs        map[string]InputTemplate    `yaml:"inputs"`
		Outputs       map[string]OutputTemplate   `yaml:"outputs"`
		InputRules    []InputRuleTemplate         `yaml:"input_rules"`
		Resources     map[string]ResourceTemplate `yaml:"resources"`
		Edges         []EdgeTemplate              `yaml:"edges"`
		resourceOrder []string
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
		From string             `yaml:"from"`
		To   string             `yaml:"to"`
		Data construct.EdgeData `yaml:"data"`
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

func (c *ConstructTemplate) ResourcesIterator() Iterator[string, ResourceTemplate] {
	return Iterator[string, ResourceTemplate]{
		source: c.Resources,
		order:  c.resourceOrder,
	}
}

func (c *ConstructTemplate) UnmarshalYAML(value *yaml.Node) error {
	type constructTemplate ConstructTemplate
	var template constructTemplate
	if err := value.Decode(&template); err != nil {
		return err
	}
	resourceOrder, _ := captureYAMLKeyOrder(value, "resources")
	template.resourceOrder = resourceOrder
	*c = ConstructTemplate(template)
	return nil
}

func (b *BindingTemplate) ResourcesIterator() Iterator[string, ResourceTemplate] {
	return Iterator[string, ResourceTemplate]{
		source: b.Resources,
		order:  b.resourceOrder,
	}
}

func (b *BindingTemplate) UnmarshalYAML(value *yaml.Node) error {
	type bindingTemplate BindingTemplate
	var template bindingTemplate
	if err := value.Decode(&template); err != nil {
		return err
	}
	resourceOrder, _ := captureYAMLKeyOrder(value, "resources")
	template.resourceOrder = resourceOrder
	*b = BindingTemplate(template)
	return nil
}

func captureYAMLKeyOrder(rootNode *yaml.Node, sectionKey string) ([]string, error) {
	var resourceOrder []string
	foundKey := false
	for i := 0; i < len(rootNode.Content); i += 2 {
		if keyNode := rootNode.Content[i]; keyNode.Value == sectionKey {
			foundKey = true
			for j := 0; j < len(rootNode.Content[i+1].Content); j += 2 {
				resourceOrder = append(resourceOrder, rootNode.Content[i+1].Content[j].Value)
			}
			break
		}
	}

	if !foundKey {
		return nil, fmt.Errorf("could not find key: %s", sectionKey)
	}

	return resourceOrder, nil
}

type Iterator[K comparable, V any] struct {
	source map[K]V
	order  []K
	index  int
}

func (r *Iterator[K, V]) Next() (K, V, bool) {
	if r.index >= len(r.order) || r.index >= len(r.source) {
		var zeroK K
		var zeroV V

		return zeroK, zeroV, false
	}

	// Get the next resource that actually exists in the map
	for _, ok := r.source[r.order[r.index]]; !ok && r.index < len(r.order); r.index++ {
		// do nothing
	}
	key := r.order[r.index]
	resource := r.source[key]

	r.index++
	return key, resource, true
}

type IterFunc[K comparable, V any] func(K, V) error

var stopIteration = fmt.Errorf("stop iteration")

func (r *Iterator[K, V]) ForEach(f IterFunc[K, V]) {
	for key, resource, ok := r.Next(); ok; key, resource, ok = r.Next() {
		if err := f(key, resource); err != nil {
			if errors.Is(err, stopIteration) {
				return
			}
		}
	}
}

func (r *Iterator[K, V]) Reset() {
	r.index = 0
}

type BindingDirection string

const (
	BindingDirectionFrom = "from"
	BindingDirectionTo   = "to"
)

func (c *ConstructTemplate) GetBindingTemplate(direction BindingDirection, other ConstructTemplateId) (BindingTemplate, error) {
	if direction == BindingDirectionFrom {
		return loadBindingTemplate(c.Id, c.Id, other)
	} else {
		return loadBindingTemplate(c.Id, other, c.Id)
	}
}
