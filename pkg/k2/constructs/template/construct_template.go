package template

import (
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"gopkg.in/yaml.v3"
	"regexp"
)

type (
	ConstructTemplate struct {
		Id            property.ConstructType      `yaml:"id"`
		Version       string                      `yaml:"version"`
		Description   string                      `yaml:"description"`
		Resources     map[string]ResourceTemplate `yaml:"resources"`
		Edges         []EdgeTemplate              `yaml:"edges"`
		Inputs        *Properties                 `yaml:"inputs"`
		Outputs       map[string]OutputTemplate   `yaml:"outputs"`
		InputRules    []InputRuleTemplate         `yaml:"input_rules"`
		resourceOrder []string
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

	OutputTemplate struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Value       any    `yaml:"value"`
	}

	InputRuleTemplate struct {
		If      string                         `yaml:"if"`
		Then    *ConditionalExpressionTemplate `yaml:"then"`
		Else    *ConditionalExpressionTemplate `yaml:"else"`
		ForEach string                         `yaml:"for_each"`
		Do      *ConditionalExpressionTemplate `yaml:"do"`
		Prefix  string                         `yaml:"prefix"`
	}

	ConditionalExpressionTemplate struct {
		Resources     map[string]ResourceTemplate `yaml:"resources"`
		Edges         []EdgeTemplate              `yaml:"edges"`
		Outputs       map[string]OutputTemplate   `yaml:"outputs"`
		Rules         []InputRuleTemplate         `yaml:"rules"`
		resourceOrder []string
	}

	ValidationTemplate struct {
		MinLength    int      `yaml:"min_length"`
		MaxLength    int      `yaml:"max_length"`
		MinValue     int      `yaml:"min_value"`
		MaxValue     int      `yaml:"max_value"`
		Pattern      string   `yaml:"pattern"`
		Enum         []string `yaml:"enum"`
		UniqueValues bool     `yaml:"unique_values"`
	}
)

var interpolationPattern = regexp.MustCompile(`\$\{([^:]+):([^}]+)}`)

func (e *EdgeTemplate) UnmarshalYAML(value *yaml.Node) error {
	// Unmarshal the edge template from a YAML node
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

func (ct *ConstructTemplate) ResourcesIterator() Iterator[string, ResourceTemplate] {
	return Iterator[string, ResourceTemplate]{
		source: ct.Resources,
		order:  ct.resourceOrder,
	}
}

func (ct *ConstructTemplate) UnmarshalYAML(value *yaml.Node) error {
	type constructTemplate ConstructTemplate
	var template constructTemplate
	if err := value.Decode(&template); err != nil {
		return err
	}
	resourceOrder, _ := captureYAMLKeyOrder(value, "resources")
	template.resourceOrder = resourceOrder
	if template.Inputs == nil {
		template.Inputs = NewProperties(nil)
	}
	if template.Resources == nil {
		template.Resources = make(map[string]ResourceTemplate)
	}

	if template.Outputs == nil {
		template.Outputs = make(map[string]OutputTemplate)
	}

	*ct = ConstructTemplate(template)
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

var StopIteration = fmt.Errorf("stop iteration")

func (r *Iterator[K, V]) ForEach(f IterFunc[K, V]) {
	for key, resource, ok := r.Next(); ok; key, resource, ok = r.Next() {
		if err := f(key, resource); err != nil {
			if errors.Is(err, StopIteration) {
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

func (ct *ConstructTemplate) GetBindingTemplate(direction BindingDirection, other property.ConstructType) (BindingTemplate, error) {
	if direction == BindingDirectionFrom {
		return LoadBindingTemplate(ct.Id, ct.Id, other)
	} else {
		return LoadBindingTemplate(ct.Id, other, ct.Id)
	}
}

func (cet *ConditionalExpressionTemplate) UnmarshalYAML(value *yaml.Node) error {
	type conditionalExpressionTemplate ConditionalExpressionTemplate

	var temp conditionalExpressionTemplate

	if err := value.Decode(&temp); err != nil {
		return err
	}

	cet.Resources = temp.Resources
	cet.Edges = temp.Edges
	cet.Outputs = temp.Outputs
	cet.Rules = temp.Rules

	resourceOrder, _ := captureYAMLKeyOrder(value, "resources")
	cet.resourceOrder = resourceOrder

	return nil
}

func (cet *ConditionalExpressionTemplate) ResourcesIterator() Iterator[string, ResourceTemplate] {
	return Iterator[string, ResourceTemplate]{
		source: cet.Resources,
		order:  cet.resourceOrder,
	}
}

func (irt *InputRuleTemplate) UnmarshalYAML(value *yaml.Node) error {
	type inputRuleTemplate InputRuleTemplate

	var temp inputRuleTemplate

	if err := value.Decode(&temp); err != nil {
		return err
	}

	if (temp.If == "" && temp.ForEach == "") || (temp.If != "" && temp.ForEach != "") {
		return fmt.Errorf("invalid InputRuleTemplate: must have either If-Then-Else or ForEach-Do")
	}

	// Check if it's an If-Then-Else structure
	if temp.If != "" {
		if temp.ForEach != "" || temp.Do != nil {
			return fmt.Errorf("invalid InputRuleTemplate: cannot mix If-Then-Else with ForEach-Do")
		}
		irt.If = temp.If
		irt.Then = temp.Then
		irt.Else = temp.Else
	} else if temp.ForEach != "" {
		// Check if it's a ForEach-Do structure
		if temp.If != "" || temp.Then != nil || temp.Else != nil {
			return fmt.Errorf("invalid InputRuleTemplate: cannot mix ForEach-Do with If-Then-Else")
		}
		irt.ForEach = temp.ForEach
		irt.Do = temp.Do
	} else {
		return fmt.Errorf("invalid InputRuleTemplate: must have either If-Then-Else or ForEach-Do")
	}

	irt.Prefix = temp.Prefix

	return nil
}

func (ct *ConstructTemplate) GetInput(path string) property.Property {
	return property.GetProperty(ct.Inputs.propertyMap, path)
}

// ForEachInput walks the input properties of a construct template,
// including nested properties, and calls the given function for each input.
// If the function returns an error, the walk will stop and return that error.
// If the function returns [ErrStopWalk], the walk will stop and return nil.
func (ct *ConstructTemplate) ForEachInput(c construct.Properties, f func(property.Property) error) error {
	return ct.Inputs.ForEach(c, f)
}
