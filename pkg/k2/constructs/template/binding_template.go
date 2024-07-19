package template

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"gopkg.in/yaml.v3"
)

type BindingTemplate struct {
	From          property.ConstructType      `yaml:"from"`
	To            property.ConstructType      `yaml:"to"`
	Priority      int                         `yaml:"priority"`
	Inputs        *Properties                 `yaml:"inputs"`
	Outputs       map[string]OutputTemplate   `yaml:"outputs"`
	InputRules    []InputRuleTemplate         `yaml:"input_rules"`
	Resources     map[string]ResourceTemplate `yaml:"resources"`
	Edges         []EdgeTemplate              `yaml:"edges"`
	resourceOrder []string
}

func (bt *BindingTemplate) GetInput(path string) property.Property {
	return property.GetProperty(bt.Inputs.propertyMap, path)
}

// ForEachInput walks the input properties of a construct template,
// including nested properties, and calls the given function for each input.
// If the function returns an error, the walk will stop and return that error.
// If the function returns [ErrStopWalk], the walk will stop and return nil.
func (bt *BindingTemplate) ForEachInput(c construct.Properties, f func(property.Property) error) error {
	return bt.Inputs.ForEach(c, f)
}

func (bt *BindingTemplate) ResourcesIterator() Iterator[string, ResourceTemplate] {
	return Iterator[string, ResourceTemplate]{
		source: bt.Resources,
		order:  bt.resourceOrder,
	}
}

func (bt *BindingTemplate) UnmarshalYAML(value *yaml.Node) error {
	type bindingTemplate BindingTemplate
	var template bindingTemplate
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

	if template.Edges == nil {
		template.Edges = make([]EdgeTemplate, 0)
	}

	if template.Outputs == nil {
		template.Outputs = make(map[string]OutputTemplate)
	}

	if template.InputRules == nil {
		template.InputRules = make([]InputRuleTemplate, 0)
	}

	*bt = BindingTemplate(template)
	return nil
}
