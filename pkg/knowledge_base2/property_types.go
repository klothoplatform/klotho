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
		SetProperty(property Property)
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

var PropertyTypeMap = map[string]PropertyType{
	"string":   &StringPropertyType{},
	"int":      &IntPropertyType{},
	"float":    &FloatPropertyType{},
	"bool":     &BoolPropertyType{},
	"resource": &ResourcePropertyType{},
	"map":      &MapPropertyType{},
	"list":     &ListPropertyType{},
}

func (p Property) IsPropertyTypeScalar() bool {
	return len(strings.Split(p.Type, "(")) == 1
}

func (p Property) getPropertyType() (PropertyType, error) {
	if p.Type == "" {
		return nil, fmt.Errorf("property %s does not have a type", p.Name)
	}
	parts := strings.Split(p.Type, "(")
	if len(parts) == 1 {
		ptype, found := PropertyTypeMap[p.Type]
		newPtype := reflect.New(reflect.TypeOf(ptype).Elem()).Interface().(PropertyType)
		if !found {
			return nil, fmt.Errorf("unknown property type '%s' for property %s", p.Type, p.Name)
		}
		newPtype.SetProperty(p)
		return newPtype, nil
	}
	args := strings.Split(strings.TrimSuffix(parts[1], ")"), ",")
	switch parts[0] {
	case "map":
		if p.Properties != nil {
			return &MapPropertyType{Property: p}, nil
		}
		if len(args) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for map property type")
		}
		return &MapPropertyType{Key: args[0], Value: args[1], Property: p}, nil
	case "list":
		if p.Properties != nil {
			return &ListPropertyType{Property: p}, nil
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for list property type")
		}
		return &ListPropertyType{Value: args[0], Property: p}, nil
	default:
		return nil, fmt.Errorf("unknown property type %s", parts[0])
	}
}

func (str *StringPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	// Here we have to try to parse to a property ref first, since a string representation of a property ref would match string parsing
	refPType := &PropertyRefPropertyType{}
	val, err := refPType.Parse(value, ctx, data)
	if err == nil {
		return val, nil
	}
	if val, ok := value.(string); ok {
		var result string
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	return nil, fmt.Errorf("invalid string value %v", value)
}

func (i *IntPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		var result int
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(int); ok {
		return val, nil
	}
	refPType := &PropertyRefPropertyType{}
	val, err := refPType.Parse(value, ctx, data)
	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid int value %v", value)
}

func (f *FloatPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
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
	refPType := &PropertyRefPropertyType{}
	val, err := refPType.Parse(value, ctx, data)
	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid float value %v", value)
}

func (b *BoolPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
	if val, ok := value.(string); ok {
		var result bool
		err := ctx.ExecuteDecode(val, data, &result)
		return result, err
	}
	if val, ok := value.(bool); ok {
		return val, nil
	}
	refPType := &PropertyRefPropertyType{}
	val, err := refPType.Parse(value, ctx, data)
	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid bool value %v", value)
}

func (r *ResourcePropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
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
	refPType := &PropertyRefPropertyType{}
	val, err := refPType.Parse(value, ctx, data)
	if err == nil {
		return val, nil
	}
	return nil, fmt.Errorf("invalid resource value %v", value)
}

func (p *PropertyRefPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {
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

func (list *ListPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {

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
			parser := PropertyTypeMap[list.Value]
			val, err := parser.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
	}
	return result, nil
}

func (m *MapPropertyType) Parse(value any, ctx ConfigTemplateContext, data ConfigTemplateData) (any, error) {

	result := map[string]any{}

	mapVal, ok := value.(map[string]any)
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

			propertyType, err := m.Property.Properties[key].getPropertyType()
			if err != nil {
				return nil, fmt.Errorf("unable to get property type for sub property %s: %w", key, err)
			} else if propertyType == nil {
				return nil, fmt.Errorf("%s is not a valid sub property", key)
			}
			propertyType.SetProperty(m.Property.Properties[key])
			val, err := propertyType.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result[key] = val

		} else {
			parser := PropertyTypeMap[keyType]
			keyVal, err := parser.Parse(key, ctx, data)
			if err != nil {
				return nil, err
			}
			parser = PropertyTypeMap[valType]
			val, err := parser.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result[keyVal.(string)] = val
		}

	}
	return result, nil
}

func (m *MapPropertyType) SetProperty(property Property) {
	m.Property = property
}

func (l *ListPropertyType) SetProperty(property Property) {
	l.Property = property
}

func (s *StringPropertyType) SetProperty(property Property) {
}

func (i *IntPropertyType) SetProperty(property Property) {
}

func (f *FloatPropertyType) SetProperty(property Property) {
}

func (b *BoolPropertyType) SetProperty(property Property) {
}

func (r *ResourcePropertyType) SetProperty(property Property) {
}

func (p *PropertyRefPropertyType) SetProperty(property Property) {
}
