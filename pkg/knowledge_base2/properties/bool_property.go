package properties

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	BoolProperty struct {
		DefaultValue *bool `json:"default_value" yaml:"default_value"`
		SharedPropertyFields
		*knowledgebase.PropertyDetails
	}
)

func (b *BoolProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(bool); ok {
		return resource.SetProperty(b.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return resource.SetProperty(b.Path, val)
	}
	return fmt.Errorf("invalid bool value %v", value)
}

func (b *BoolProperty) AppendProperty(resource *construct.Resource, value any) error {
	return b.SetProperty(resource, value)
}

func (b *BoolProperty) RemoveProperty(resource *construct.Resource, value any) error {
	delete(resource.Properties, b.Path)
	return nil
}

func (b *BoolProperty) Clone() knowledgebase.Property {
	return &BoolProperty{
		DefaultValue: b.DefaultValue,
		SharedPropertyFields: SharedPropertyFields{
			DefaultValueTemplate: b.DefaultValueTemplate,
			ValidityChecks:       b.ValidityChecks,
		},
		PropertyDetails: &knowledgebase.PropertyDetails{
			Name:                  b.Name,
			Path:                  b.Path,
			Required:              b.Required,
			ConfigurationDisabled: b.ConfigurationDisabled,
			DeployTime:            b.DeployTime,
			OperationalRule:       b.OperationalRule,
			Namespace:             b.Namespace,
		},
	}
}

func (b *BoolProperty) Details() *knowledgebase.PropertyDetails {
	return b.PropertyDetails
}

func (b *BoolProperty) GetDefaultValue(ctx knowledgebase.DynamicValueContext, data knowledgebase.DynamicValueData) (any, error) {
	if b.DefaultValue != nil {
		return b.DefaultValue, nil
	} else if b.DefaultValueTemplate != nil {
		var result bool
		err := ctx.ExecuteTemplateDecode(b.DefaultValueTemplate, data, &result)
		if err != nil {
			return decodeAsPropertyRef(b.DefaultValueTemplate, ctx, data)
		}
		return result, nil
	}
	return nil, nil
}

func (b *BoolProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		var result bool
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(bool); ok {
		return val, nil
	}
	val, err := ParsePropertyRef(value, ctx, data)

	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid bool value %v", value)
}

func (b *BoolProperty) ZeroValue() any {
	return false
}

func (b *BoolProperty) Contains(value any, contains any) bool {
	return false
}

func (b *BoolProperty) Type() string {
	return "bool"
}

func (b *BoolProperty) Validate(value any, properties construct.Properties) error {
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("invalid bool value %v", value)
	}
	return nil
}

func (b *BoolProperty) SubProperties() map[string]knowledgebase.Property {
	return nil
}
