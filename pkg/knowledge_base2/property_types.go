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
		ZeroValue() any
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

var PropertyTypeMap = map[string]func() PropertyType{
	"string":   func() PropertyType { return &StringPropertyType{} },
	"int":      func() PropertyType { return &IntPropertyType{} },
	"float":    func() PropertyType { return &FloatPropertyType{} },
	"bool":     func() PropertyType { return &BoolPropertyType{} },
	"resource": func() PropertyType { return &ResourcePropertyType{} },
	"map":      func() PropertyType { return &MapPropertyType{} },
	"list":     func() PropertyType { return &ListPropertyType{} },
}

func (p Property) IsPropertyTypeScalar() bool {
	return len(strings.Split(p.Type, "(")) == 1
}

func (p Property) PropertyType() (PropertyType, error) {
	if p.Type == "" {
		return nil, fmt.Errorf("property %s does not have a type", p.Name)
	}
	parts := strings.Split(p.Type, "(")
	if len(parts) == 1 {
		ptypeGen, found := PropertyTypeMap[p.Type]
		if !found {
			return nil, fmt.Errorf("unknown property type '%s' for property %s", p.Type, p.Name)
		}
		newPtype := ptypeGen()
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
			parserGen, found := PropertyTypeMap[list.Value]
			if !found {
				return nil, fmt.Errorf("invalid list property type %s", list.Property.Name)
			}
			parser := parserGen()
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

			propertyType, err := m.Property.Properties[key].PropertyType()
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
			parserGen, found := PropertyTypeMap[keyType]
			if !found {
				return nil, fmt.Errorf("invalid map property type %s", m.Property.Name)
			}
			parser := parserGen()
			keyVal, err := parser.Parse(key, ctx, data)
			if err != nil {
				return nil, err
			}
			parserGen = PropertyTypeMap[valType]
			if !found {
				return nil, fmt.Errorf("invalid map property type %s", m.Property.Name)
			}
			parser = parserGen()
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

func (m *MapPropertyType) ZeroValue() any {
	keyZero := PropertyTypeMap[m.Key]().ZeroValue()
	valZero := PropertyTypeMap[m.Value]().ZeroValue()
	return reflect.MakeMap(
		reflect.MapOf(reflect.TypeOf(keyZero), reflect.TypeOf(valZero))).
		Interface()
}

func (l *ListPropertyType) ZeroValue() any {
	elemZero := PropertyTypeMap[l.Value]().ZeroValue()
	return reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(elemZero)), 0, 0).Interface()
}

func (s *StringPropertyType) ZeroValue() any {
	return ""
}

func (i *IntPropertyType) ZeroValue() any {
	return 0
}

func (f *FloatPropertyType) ZeroValue() any {
	return 0.0
}

func (b *BoolPropertyType) ZeroValue() any {
	return false
}

func (r *ResourcePropertyType) ZeroValue() any {
	return construct.ResourceId{}
}

func (p *PropertyRefPropertyType) ZeroValue() any {
	return construct.PropertyRef{}
}
