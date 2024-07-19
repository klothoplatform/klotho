package properties

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
)

type (
	FloatProperty struct {
		MinValue *float64
		MaxValue *float64
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (f *FloatProperty) SetProperty(properties construct.Properties, value any) error {
	switch val := value.(type) {
	case float64:
		return properties.SetProperty(f.Path, val)
	case construct.PropertyRef:
		return properties.SetProperty(f.Path, val)
	case float32:
		return properties.SetProperty(f.Path, float64(val))
	case int:
		return properties.SetProperty(f.Path, float64(val))
	default:
		return fmt.Errorf("invalid float value %v", value)
	}
}

func (f *FloatProperty) AppendProperty(properties construct.Properties, value any) error {
	return f.SetProperty(properties, value)
}

func (f *FloatProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(f.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return properties.RemoveProperty(f.Path, value)

}

func (f *FloatProperty) Details() *property.PropertyDetails {
	return &f.PropertyDetails
}

func (f *FloatProperty) Clone() property.Property {
	clone := *f
	return &clone
}

func (f *FloatProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if f.DefaultValue == nil {
		return nil, nil
	}
	return f.Parse(f.DefaultValue, ctx, data)
}

func (f *FloatProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	if val, ok := value.(string); ok {
		var result float32
		err := ctx.ExecuteUnmarshal(val, data, &result)
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
	return nil, fmt.Errorf("invalid float value %v", value)
}

func (f *FloatProperty) ZeroValue() any {
	return 0.0
}

func (f *FloatProperty) Contains(value any, contains any) bool {
	return false
}

func (f *FloatProperty) Type() string {
	return "float"
}

func (f *FloatProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if f.Required {
			return fmt.Errorf(property.ErrRequiredProperty, f.Path)
		}
		return nil
	}
	floatVal, ok := value.(float64)
	if !ok {
		return fmt.Errorf("invalid float value %v", value)
	}
	if f.MinValue != nil && floatVal < *f.MinValue {
		return fmt.Errorf("float value %f is less than lower bound %f", value, *f.MinValue)
	}
	if f.MaxValue != nil && floatVal > *f.MaxValue {
		return fmt.Errorf("float value %f is greater than upper bound %f", value, *f.MaxValue)
	}
	return nil
}

func (f *FloatProperty) SubProperties() property.PropertyMap {
	return nil
}
