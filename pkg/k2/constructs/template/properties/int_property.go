package properties

import (
	"errors"
	"fmt"
	"math"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
)

type (
	IntProperty struct {
		MinValue *int
		MaxValue *int
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (i *IntProperty) SetProperty(properties construct.Properties, value any) error {
	if val, ok := value.(int); ok {
		return properties.SetProperty(i.Path, val)
	} else if val, ok := value.(construct.PropertyRef); ok {
		return properties.SetProperty(i.Path, val)
	}
	return fmt.Errorf("invalid int value %v", value)
}

func (i *IntProperty) AppendProperty(properties construct.Properties, value any) error {
	return i.SetProperty(properties, value)
}

func (i *IntProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(i.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return properties.RemoveProperty(i.Path, value)
}

func (i *IntProperty) Details() *property.PropertyDetails {
	return &i.PropertyDetails
}

func (i *IntProperty) Clone() property.Property {
	clone := *i
	return &clone
}

func (i *IntProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if i.DefaultValue == nil {
		return nil, nil
	}
	return i.Parse(i.DefaultValue, ctx, data)
}

func (i *IntProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {

	if val, ok := value.(string); ok {
		var result int
		err := ctx.ExecuteUnmarshal(val, data, &result)
		return result, err
	}
	if val, ok := value.(int); ok {
		return val, nil
	}
	EPSILON := 0.0000001
	if val, ok := value.(float32); ok {
		ival := int(val)
		if math.Abs(float64(val)-float64(ival)) > EPSILON {
			return 0, fmt.Errorf("cannot convert non-integral float to int: %f", val)
		}
		return int(val), nil

	} else if val, ok := value.(float64); ok {
		ival := int(val)
		if math.Abs(val-float64(ival)) > EPSILON {
			return 0, fmt.Errorf("cannot convert non-integral float to int: %f", val)
		}
		return int(val), nil
	}
	return nil, fmt.Errorf("invalid int value %v", value)
}

func (i *IntProperty) ZeroValue() any {
	return 0
}

func (i *IntProperty) Contains(value any, contains any) bool {
	return false
}

func (i *IntProperty) Type() string {
	return "int"
}

func (i *IntProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if i.Required {
			return fmt.Errorf(property.ErrRequiredProperty, i.Path)
		}
		return nil
	}
	intVal, ok := value.(int)
	if !ok {
		return fmt.Errorf("invalid int value %v", value)
	}
	if i.MinValue != nil && intVal < *i.MinValue {
		return fmt.Errorf("int value %v is less than lower bound %d", value, *i.MinValue)
	}
	if i.MaxValue != nil && intVal > *i.MaxValue {
		return fmt.Errorf("int value %v is greater than upper bound %d", value, *i.MaxValue)
	}
	return nil
}

func (i *IntProperty) SubProperties() property.PropertyMap {
	return nil
}
