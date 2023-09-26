package knowledgebase2

import (
	"fmt"
	"reflect"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	PropertyType interface {
		Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error)
	}

	MapPropertyType struct {
		Property Property
		Key      PropertyTypes
		Value    PropertyTypes
	}

	ListPropertyType struct {
		Property Property
		Value    PropertyTypes
	}

	ScalarPropertyType struct {
		Property Property
		Type     PropertyTypes
	}

	PropertyTypes string
)

const (
	// StringPropertyType is a string property type
	StringPropertyType PropertyTypes = "string"
	// IntPropertyType is an int property type
	IntPropertyType PropertyTypes = "int"
	// FloatPropertyType is a float property type
	FloatPropertyType PropertyTypes = "float"
	// BoolPropertyType is a bool property type
	BoolPropertyType PropertyTypes = "bool"
	// ResourcePropertyType is a resource property type
	ResourcePropertyType PropertyTypes = "resource"
	// PropertyReferencePropertyType is a property reference property type
	PropertyReferencePropertyType PropertyTypes = "property_reference"
	// ObjectPropertyType is an object property type
	ObjectPropertyType PropertyTypes = "object"
)

func (p Property) getPropertyType() (PropertyType, error) {
	parts := strings.Split(p.Type, "(")
	if len(parts) == 1 {
		return ScalarPropertyType{Type: PropertyTypes(p.Type), Property: p}, nil
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
		return MapPropertyType{Key: PropertyTypes(args[0]), Value: PropertyTypes(args[1]), Property: p}, nil
	case "list":
		if p.Properties != nil {
			return ListPropertyType{Property: p}, nil
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for list property type")
		}
		return ListPropertyType{Value: PropertyTypes(args[0]), Property: p}, nil
	default:
		return nil, fmt.Errorf("unknown property type %s", parts[0])
	}
}

func (ctx ConfigTemplateContext) parsePropertyValue(propertyType PropertyTypes, value any, data ConfigTemplateData) (any, error) {
	switch propertyType {
	case StringPropertyType:
		if val, ok := value.(string); ok {
			var result string
			err := ctx.ExecuteDecode(val, data, &result)
			return result, err
		}
		return nil, fmt.Errorf("invalid string value %v", value)
	case IntPropertyType:
		if val, ok := value.(string); ok {
			var result int
			err := ctx.ExecuteDecode(val, data, &result)
			return result, err
		}
		if val, ok := value.(int); ok {
			return val, nil
		}
	case FloatPropertyType:
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
		return nil, fmt.Errorf("invalid float value %v", value)
	case BoolPropertyType:
		if val, ok := value.(string); ok {
			var result bool
			err := ctx.ExecuteDecode(val, data, &result)
			return result, err
		}
		if val, ok := value.(bool); ok {
			return val, nil
		}
		return nil, fmt.Errorf("invalid bool value %v", value)
	case ResourcePropertyType:
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
	case PropertyReferencePropertyType:
		if val, ok := value.(string); ok {
			result := construct.PropertyRef{}
			err := ctx.ExecuteDecode(val, data, &result)
			return result, err
		}
		if val, ok := value.(map[string]interface{}); ok {
			id, err := ctx.parsePropertyValue(ResourcePropertyType, val["resource"], data)
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
	return nil, fmt.Errorf("unknown scalar property type %s", propertyType)
}

func (scalar ScalarPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	return ctx.parsePropertyValue(scalar.Type, value, data)
}

func (list ListPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {

	var result []any
	reflectVal := reflect.ValueOf(value)
	if reflectVal.Kind() != reflect.Array || reflectVal.Kind() != reflect.Slice {
		return nil, fmt.Errorf("invalid list value %v", value)
	}

	for i := 0; i < reflectVal.Len(); i++ {
		if list.Property.Properties != nil {
			m := MapPropertyType{Property: list.Property}
			val, err := m.Parse(reflectVal.Index(i).Interface(), ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		} else {
			val, err := ctx.parsePropertyValue(list.Value, reflectVal.Index(i).Interface(), data)
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

	reflectVal := reflect.ValueOf(value)
	if reflectVal.Kind() != reflect.Map {
		return nil, fmt.Errorf("invalid map value %v", value)
	}
	for _, key := range reflectVal.MapKeys() {
		keyType := m.Key
		valType := m.Value
		// If we are an object with sub properties then we know that we need to get the type of our sub properties to determine how we are parsed into a value
		if m.Property.Properties != nil {
			if m.Key != "" || m.Value != "" {
				return nil, fmt.Errorf("invalid map property type %s", m.Property.Name)
			}

			PropertyType, err := m.Property.Properties[key.Interface().(string)].getPropertyType()
			if err != nil {
				return nil, err
			}
			val, err := PropertyType.Parse(reflectVal.MapIndex(key).Interface(), ctx, data)
			if err != nil {
				return nil, err
			}
			result[key.Interface()] = val

		} else {
			keyVal, err := ctx.parsePropertyValue(keyType, key.Interface(), data)
			if err != nil {
				return nil, err
			}

			val, err := ctx.parsePropertyValue(valType, reflectVal.MapIndex(key).Interface(), data)
			if err != nil {
				return nil, err
			}
			result[keyVal] = val
		}

	}
	return result, nil
}
