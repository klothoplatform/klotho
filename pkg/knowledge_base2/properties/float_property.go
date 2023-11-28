package properties

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	FloatProperty struct {
		LowerBound   *float64 `yaml:"lower_bound"`
		UpperBound   *float64 `yaml:"upper_bound"`
		DefaultValue *float64 `json:"default_value" yaml:"default_value"`
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (f *FloatProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(float64); ok {
		return resource.SetProperty(f.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return resource.SetProperty(f.Path, val)
	}
	return fmt.Errorf("invalid float value %v", value)
}

func (f *FloatProperty) AppendProperty(resource *construct.Resource, value any) error {
	return f.SetProperty(resource, value)
}

func (f *FloatProperty) RemoveProperty(resource *construct.Resource, value any) error {
	delete(resource.Properties, f.Path)
	return nil
}

func (f *FloatProperty) Details() *knowledgebase.PropertyDetails {
	return f.PropertyDetails
}

func (f *FloatProperty) Clone() knowledgebase.Property {
	return &FloatProperty{
		LowerBound:   f.LowerBound,
		UpperBound:   f.UpperBound,
		DefaultValue: f.DefaultValue,
		SharedPropertyFields: SharedPropertyFields{
			DefaultValueTemplate: f.DefaultValueTemplate,
			ValidityChecks:       f.ValidityChecks,
		},
		PropertyDetails: &knowledgebase.PropertyDetails{
			Name:                  f.Name,
			Path:                  f.Path,
			Required:              f.Required,
			ConfigurationDisabled: f.ConfigurationDisabled,
			DeployTime:            f.DeployTime,
			OperationalRule:       f.OperationalRule,
			Namespace:             f.Namespace,
		},
	}
}

func (f *FloatProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if f.DefaultValue != nil {
		return f.DefaultValue, nil
	} else if f.DefaultValueTemplate != nil {
		var result float64
		err := ctx.ExecuteTemplateDecode(f.DefaultValueTemplate, data, &result)
		if err != nil {
			return decodeAsPropertyRef(f.DefaultValueTemplate, ctx, data)
		}
		return result, nil
	}
	return nil, nil
}

func (f *FloatProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		var result float32
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(float32); ok {
		return val, nil
	}
	if val, ok := value.(float64); ok {
		return val, nil
	}
	if val, ok := value.(int); ok {
		return float64(val), nil
	}
	val, err := ParsePropertyRef(value, ctx, data)
	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid float value %v", value)
}

func (f *FloatProperty) ZeroValue() any {
	return 0.0
}

func (f *FloatProperty) Contains(value any, contains any) bool {
	return false
}

func (f *FloatProperty) SetPath(path string) {
	f.Path = path
}

func (f *FloatProperty) PropertyName() string {
	return f.Name
}

func (f *FloatProperty) Type() string {
	return "float"
}

func (f *FloatProperty) IsRequired() bool {
	return f.Required
}

func (f *FloatProperty) Validate(value any, properties construct.Properties) error {
	floatVal, ok := value.(float64)
	if !ok {
		return fmt.Errorf("invalid int value %v", value)
	}
	if f.LowerBound != nil && floatVal < *f.LowerBound {
		return fmt.Errorf("int value %v is less than lower bound %d", value, *f.LowerBound)
	}
	if f.UpperBound != nil && floatVal > *f.UpperBound {
		return fmt.Errorf("int value %v is greater than upper bound %d", value, *f.UpperBound)
	}
	return nil
}

func (f *FloatProperty) SubProperties() map[string]knowledgebase.Property {
	return nil
}
