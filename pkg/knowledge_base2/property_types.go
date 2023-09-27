package knowledgebase2

import (
	"fmt"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	PropertyType interface {
		Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error)
	}

	MapPropertyType struct {
		Property Property
		Key      string
		Value    string
	}

	ListPropertyType struct {
		Property Property
		Value    string
	}

	StringPropertyType      struct{}
	IntPropertyType         struct{}
	FloatPropertyType       struct{}
	BoolPropertyType        struct{}
	ResourcePropertyType    struct{}
	PropertyRefPropertyType struct{}
)

var ScalarPropertyMap = map[string]PropertyType{
	"string":      StringPropertyType{},
	"int":         IntPropertyType{},
	"float":       FloatPropertyType{},
	"bool":        BoolPropertyType{},
	"resource":    ResourcePropertyType{},
	"propertyref": PropertyRefPropertyType{},
}

func (p Property) IsPropertyTypeScalar() bool {
	return len(strings.Split(p.Type, "(")) == 1
}

func (p Property) getPropertyType() (PropertyType, error) {
	parts := strings.Split(p.Type, "(")
	if len(parts) == 1 {
		return ScalarPropertyMap[p.Type], nil
	}
	args := strings.Split(strings.TrimSuffix(parts[1], ")"), ",")
	switch parts[0] {
	case "map":
		if p.Properties != nil {
			return MapPropertyType{Property: p}, nil
		}
		if len(args) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for map property type")
		}
		return MapPropertyType{Key: args[0], Value: args[1], Property: p}, nil
	case "list":
		if p.Properties != nil {
			return ListPropertyType{Property: p}, nil
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for list property type")
		}
		return ListPropertyType{Value: args[0], Property: p}, nil
	default:
		return nil, fmt.Errorf("unknown property type %s", parts[0])
	}
}

func (str StringPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		var result string
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	return nil, fmt.Errorf("invalid string value %v", value)
}

func (i IntPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		var result int
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(int); ok {
		return val, nil
	}
	return nil, fmt.Errorf("invalid int value %v", value)
}

func (f FloatPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
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
	return nil, fmt.Errorf("invalid float value %v", value)
}

func (b BoolPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		var result bool
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(bool); ok {
		return val, nil
	}
	return nil, fmt.Errorf("invalid bool value %v", value)
}

func (r ResourcePropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		return ctx.ExecuteDecodeAsResourceId(val, data)
	}
	if val, ok := value.(map[string]interface{}); ok {
		id := construct.ResourceId{
			Type:     val["type"].(string),
			Name:     val["name"].(string),
			Provider: val["provider"].(string),
		}
		if namespace, ok := val["namespace"]; ok {
			id.Namespace = namespace.(string)
		}
		return id, nil
	}
	if val, ok := value.(construct.ResourceId); ok {
		return val, nil
	}
	return nil, fmt.Errorf("invalid resource value %v", value)
}

func (p PropertyRefPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		result := construct.PropertyRef{}
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(map[string]interface{}); ok {
		rp := ResourcePropertyType{}
		id, err := rp.Parse(val["resource"], ctx, data)
		if err != nil {
			return nil, err
		}
		return construct.PropertyRef{
			Property: val["property"].(string),
			Resource: id.(construct.ResourceId),
		}, nil
	}
	if val, ok := value.(construct.PropertyRef); ok {
		return val, nil
	}
	return nil, fmt.Errorf("invalid property reference value %v", value)
}

func (list ListPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {

	var result []any
	val, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid list value %v", value)
	}

	for _, v := range val {
		if list.Property.Properties != nil {
			m := MapPropertyType{Property: list.Property}
			val, err := m.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		} else {
			parser := ScalarPropertyMap[list.Value]
			val, err := parser.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}

	}
	return result, nil
}

func (m MapPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {

	result := map[any]any{}

	mapVal, ok := value.(map[any]any)
	if !ok {
		return nil, fmt.Errorf("invalid map value %v", value)
	}

	for key, v := range mapVal {
		keyType := m.Key
		valType := m.Value
		// If we are an object with sub properties then we know that we need to get the type of our sub properties to determine how we are parsed into a value
		if m.Property.Properties != nil {
			if m.Key != "" || m.Value != "" {
				return nil, fmt.Errorf("invalid map property type %s", m.Property.Name)
			}

			propertyType, err := m.Property.Properties[key.(string)].getPropertyType()
			if err != nil {
				return nil, err
			} else if propertyType == nil {
				return nil, fmt.Errorf("%s is not a valid sub property", key.(string))
			}
			val, err := propertyType.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result[key] = val

		} else {
			parser := ScalarPropertyMap[keyType]
			keyVal, err := parser.Parse(key, ctx, data)
			if err != nil {
				return nil, err
			}
			parser = ScalarPropertyMap[valType]
			val, err := parser.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result[keyVal] = val
		}

	}
	return result, nil
}
