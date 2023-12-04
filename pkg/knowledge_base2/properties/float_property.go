package properties

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	FloatProperty struct {
		MinValue *float64
		MaxValue *float64
		SharedPropertyFields
		knowledgebase.PropertyDetails
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
	propVal, err := resource.GetProperty(f.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return resource.RemoveProperty(f.Path, value)

}

func (f *FloatProperty) Details() *knowledgebase.PropertyDetails {
	return &f.PropertyDetails
}

func (f *FloatProperty) Clone() knowledgebase.Property {
	clone := *f
	return &clone
}

func (f *FloatProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if f.DefaultValue == nil {
		return nil, nil
	}
	return f.Parse(f.DefaultValue, ctx, data)
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

func (f *FloatProperty) Type() string {
	return "float"
}

func (f *FloatProperty) Validate(resource *construct.Resource, value any) error {
	floatVal, ok := value.(float64)
	if !ok {
		return fmt.Errorf("invalid int value %v", value)
	}
	if f.MinValue != nil && floatVal < *f.MinValue {
		return fmt.Errorf("int value %f is less than lower bound %f", value, *f.MinValue)
	}
	if f.MaxValue != nil && floatVal > *f.MaxValue {
		return fmt.Errorf("int value %f is greater than upper bound %f", value, *f.MaxValue)
	}
	return nil
}

func (f *FloatProperty) SubProperties() knowledgebase.Properties {
	return nil
}
