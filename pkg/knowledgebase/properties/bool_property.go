package properties

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	BoolProperty struct {
		SharedPropertyFields
		knowledgebase.PropertyDetails
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
	propVal, err := resource.GetProperty(b.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return resource.RemoveProperty(b.Path, value)
}

func (b *BoolProperty) Clone() knowledgebase.Property {
	clone := *b
	return &clone
}

func (b *BoolProperty) Details() *knowledgebase.PropertyDetails {
	return &b.PropertyDetails
}

func (b *BoolProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if b.DefaultValue == nil {
		return nil, nil
	}
	return b.Parse(b.DefaultValue, ctx, data)
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

func (b *BoolProperty) Validate(resource *construct.Resource, value any, ctx knowledgebase.DynamicContext) error {
	if value == nil {
		if b.Required {
			return fmt.Errorf(knowledgebase.ErrRequiredProperty, b.Path, resource.ID)
		}
		return nil
	}
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("invalid bool value %v", value)
	}
	return nil
}

func (b *BoolProperty) SubProperties() knowledgebase.Properties {
	return nil
}
