package properties

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
)

type (
	BoolProperty struct {
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (b *BoolProperty) SetProperty(properties construct.Properties, value any) error {
	if val, ok := value.(bool); ok {
		return properties.SetProperty(b.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return properties.SetProperty(b.Path, val)
	}
	return fmt.Errorf("invalid bool value %v", value)
}

func (b *BoolProperty) AppendProperty(properties construct.Properties, value any) error {
	return b.SetProperty(properties, value)
}

func (b *BoolProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(b.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return properties.RemoveProperty(b.Path, value)
}

func (b *BoolProperty) Clone() property.Property {
	clone := *b
	return &clone
}

func (b *BoolProperty) Details() *property.PropertyDetails {
	return &b.PropertyDetails
}

func (b *BoolProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if b.DefaultValue == nil {
		return nil, nil
	}
	return b.Parse(b.DefaultValue, ctx, data)
}

func (b *BoolProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	if val, ok := value.(string); ok {
		var result bool
		err := ctx.ExecuteUnmarshal(val, data, &result)
		return result, err
	}
	if val, ok := value.(bool); ok {
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

func (b *BoolProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if b.Required {
			return fmt.Errorf(property.ErrRequiredProperty, b.Path)
		}
		return nil
	}
	if _, ok := value.(bool); !ok {
		return fmt.Errorf("invalid bool value %v", value)
	}
	return nil
}

func (b *BoolProperty) SubProperties() property.PropertyMap {
	return nil
}
