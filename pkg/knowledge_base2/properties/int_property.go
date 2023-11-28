package properties

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	IntProperty struct {
		LowerBound   *int `yaml:"lower_bound"`
		UpperBound   *int `yaml:"upper_bound"`
		DefaultValue *int `json:"default_value" yaml:"default_value"`
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (i *IntProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(int); ok {
		return resource.SetProperty(i.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return resource.SetProperty(i.Path, val)
	}
	return fmt.Errorf("invalid int value %v", value)
}

func (i *IntProperty) AppendProperty(resource *construct.Resource, value any) error {
	return i.SetProperty(resource, value)
}

func (i *IntProperty) RemoveProperty(resource *construct.Resource, value any) error {
	delete(resource.Properties, i.Path)
	return nil
}

func (i *IntProperty) Details() *knowledgebase.PropertyDetails {
	return i.PropertyDetails
}

func (i *IntProperty) Clone() knowledgebase.Property {
	return &IntProperty{
		LowerBound:   i.LowerBound,
		UpperBound:   i.UpperBound,
		DefaultValue: i.DefaultValue,
		SharedPropertyFields: SharedPropertyFields{
			DefaultValueTemplate: i.DefaultValueTemplate,
			ValidityChecks:       i.ValidityChecks,
		},
		PropertyDetails: &knowledgebase.PropertyDetails{
			Name:                  i.Name,
			Path:                  i.Path,
			Required:              i.Required,
			ConfigurationDisabled: i.ConfigurationDisabled,
			DeployTime:            i.DeployTime,
			OperationalRule:       i.OperationalRule,
			Namespace:             i.Namespace,
		},
	}
}

func (i *IntProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if i.DefaultValue != nil {
		return i.DefaultValue, nil
	} else if i.DefaultValueTemplate != nil {
		var result int
		err := ctx.ExecuteTemplateDecode(i.DefaultValueTemplate, data, &result)
		if err != nil {
			return decodeAsPropertyRef(i.DefaultValueTemplate, ctx, data)
		}
		return result, nil
	}
	return nil, nil
}

func (i *IntProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		var result int
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(int); ok {
		return val, nil
	}
	val, err := ParsePropertyRef(value, ctx, data)
	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid int value %v", value)
}

func (i *IntProperty) ZeroValue() any {
	return 0
}

func (i *IntProperty) Contains(value any, contains any) bool {
	return false
}

func (i *IntProperty) SetPath(path string) {
	i.Path = path
}

func (i *IntProperty) PropertyName() string {
	return i.Name
}

func (i *IntProperty) Type() string {
	return "int"
}

func (i *IntProperty) IsRequired() bool {
	return i.Required
}

func (i *IntProperty) Validate(value any, properties construct.Properties) error {
	intVal, ok := value.(int)
	if !ok {
		return fmt.Errorf("invalid int value %v", value)
	}
	if i.LowerBound != nil && intVal < *i.LowerBound {
		return fmt.Errorf("int value %v is less than lower bound %d", value, *i.LowerBound)
	}
	if i.UpperBound != nil && intVal > *i.UpperBound {
		return fmt.Errorf("int value %v is greater than upper bound %d", value, *i.UpperBound)
	}
	return nil
}

func (i *IntProperty) SubProperties() map[string]knowledgebase.Property {
	return nil
}