package properties

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	IntProperty struct {
		MinValue *int
		MaxValue *int
		SharedPropertyFields
		knowledgebase.PropertyDetails
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
	propVal, err := resource.GetProperty(i.Path)
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	return resource.RemoveProperty(i.Path, value)
}

func (i *IntProperty) Details() *knowledgebase.PropertyDetails {
	return &i.PropertyDetails
}

func (i *IntProperty) Clone() knowledgebase.Property {
	clone := *i
	return &clone
}

func (i *IntProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if i.DefaultValue == nil {
		return nil, nil
	}
	return i.Parse(i.DefaultValue, ctx, data)
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
	if val, ok := value.(float32); ok {
		return int(val), nil
	} else if val, ok := value.(float64); ok {
		return int(val), nil
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

func (i *IntProperty) Type() string {
	return "int"
}

func (i *IntProperty) Validate(resource *construct.Resource, value any, ctx knowledgebase.DynamicContext) error {
	if value == nil {
		if i.Required {
			return fmt.Errorf(knowledgebase.ErrRequiredProperty, i.Path, resource.ID)
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

func (i *IntProperty) SubProperties() knowledgebase.Properties {
	return nil
}
