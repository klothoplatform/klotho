package knowledgebase2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	PropertyType interface {
		Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error)
		SetProperty(property *Property)
		ZeroValue() any
	}

	MapPropertyType struct {
		Property *Property
		Key      string
		Value    string
	}

	ListPropertyType struct {
		Property *Property
		Value    string
	}
	SetPropertyType struct {
		Property *Property
		Value    string
	}

	StringPropertyType   struct{}
	IntPropertyType      struct{}
	FloatPropertyType    struct{}
	BoolPropertyType     struct{}
	ResourcePropertyType struct {
		Value construct.ResourceId
	}
	PropertyRefPropertyType struct{}
	AnyPropertyType         struct{}
)

var PropertyTypeMap = map[string]func(val string, property *Property) (PropertyType, error){
	"string": func(val string, property *Property) (PropertyType, error) { return &StringPropertyType{}, nil },
	"int":    func(val string, property *Property) (PropertyType, error) { return &IntPropertyType{}, nil },
	"float":  func(val string, property *Property) (PropertyType, error) { return &FloatPropertyType{}, nil },
	"bool":   func(val string, property *Property) (PropertyType, error) { return &BoolPropertyType{}, nil },
	"resource": func(val string, property *Property) (PropertyType, error) {
		id := construct.ResourceId{}
		err := id.UnmarshalText([]byte(val))
		if err != nil {
			return nil, fmt.Errorf("invalid resource id for property type %s: %w", val, err)
		}
		return &ResourcePropertyType{Value: id}, nil
	},
	"map": func(val string, property *Property) (PropertyType, error) {
		args := strings.Split(val, ",")
		if property.Properties != nil {
			return &MapPropertyType{Property: property}, nil
		}
		if len(args) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for map property type")
		}
		return &MapPropertyType{Key: args[0], Value: args[1], Property: property}, nil
	},
	"list": func(val string, p *Property) (PropertyType, error) {
		if p.Properties != nil {
			return &ListPropertyType{Property: p}, nil
		}
		return &ListPropertyType{Value: val, Property: p}, nil
	},
	"set": func(val string, p *Property) (PropertyType, error) {
		if p.Properties != nil {
			return &SetPropertyType{Property: p}, nil
		}
		return &SetPropertyType{Value: val, Property: p}, nil
	},
	"any": func(val string, property *Property) (PropertyType, error) { return &AnyPropertyType{}, nil },
}

func (p Properties) Clone() Properties {
	newProps := make(Properties, len(p))
	for k, v := range p {
		newProps[k] = v.Clone()
	}
	return newProps
}

func (p *Property) Clone() *Property {
	newProps := make(map[string]*Property, len(p.Properties))
	for k, v := range p.Properties {
		newProps[k] = v.Clone()
	}
	return &Property{
		Name:                  p.Name,
		Path:                  p.Path,
		Type:                  p.Type,
		DefaultValue:          p.DefaultValue,
		Properties:            newProps,
		Namespace:             p.Namespace,
		Required:              p.Required,
		ConfigurationDisabled: p.ConfigurationDisabled,
		OperationalRule:       p.OperationalRule,
	}
}

func (p Property) IsPropertyTypeScalar() bool {
	return !collectionutil.Contains([]string{"map", "list", "set"}, strings.Split(p.Type, "(")[0])
}

func (p Property) ModelType() *string {
	typeString := strings.TrimSuffix(strings.TrimPrefix(p.Type, "list("), ")")
	parts := strings.Split(typeString, "(")
	if parts[0] != "model" {
		return nil
	}
	if len(parts) == 1 {
		return &p.Name
	}
	if len(parts) != 2 {
		return nil
	}
	modelType := strings.TrimSuffix(parts[1], ")")
	return &modelType
}

func (p *Property) PropertyType() (PropertyType, error) {
	if p.Type == "" {
		return nil, fmt.Errorf("property %s does not have a type", p.Name)
	}
	parts := strings.Split(p.Type, "(")
	ptypeGen, found := PropertyTypeMap[parts[0]]
	if !found {
		return nil, fmt.Errorf("unknown property type '%s' for property %s", p.Type, p.Name)
	}
	val := strings.TrimSuffix(strings.Join(parts[1:], "("), ")")
	newPtype, err := ptypeGen(val, p)
	if err != nil {
		return nil, fmt.Errorf("unable to create property type for property %s: %w", p.Name, err)
	}
	newPtype.SetProperty(p)
	return newPtype, nil
}

func (str *StringPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
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

func (i *IntPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
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

func (f *FloatPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
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

func (b *BoolPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
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

func (r *ResourcePropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
	if val, ok := value.(string); ok {
		id, err := ExecuteDecodeAsResourceId(ctx, val, data)
		if !id.IsZero() && !r.Value.Matches(id) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.Value)
		}
		return id, err
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
		if !r.Value.Matches(id) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.Value)
		}
		return id, nil
	}
	if val, ok := value.(construct.ResourceId); ok {
		if !r.Value.Matches(val) {
			return nil, fmt.Errorf("resource value %v does not match type %s", value, r.Value)
		}
		return val, nil
	}
	refPType := &PropertyRefPropertyType{}
	val, err := refPType.Parse(value, ctx, data)
	if err == nil {
		if ptype, ok := val.(construct.PropertyRef); ok {
			if !r.Value.Matches(ptype.Resource) {
				return nil, fmt.Errorf("resource value %v does not match type %s", value, r.Value)
			}
		}
		return val, nil
	}
	return nil, fmt.Errorf("invalid resource value %v", value)
}

func (p *PropertyRefPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
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

func (list *ListPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {

	var result []any
	val, ok := value.([]any)
	if !ok {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			var result []any
			err := ctx.ExecuteDecode(strVal, data, &result)
			return result, err
		}
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
			tempProp := &Property{Type: list.Value}
			parser, err := tempProp.PropertyType()
			if err != nil {
				return nil, fmt.Errorf("invalid value type for list property type %s", list.Value)
			}
			val, err := parser.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result = append(result, val)
		}
	}
	return result, nil
}

func (s *SetPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
	var result = set.HashedSet[string, any]{
		Hasher: func(s any) string {
			return fmt.Sprintf("%v", s)
		},
	}
	val, ok := value.([]any)
	if !ok {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			var result []any
			err := ctx.ExecuteDecode(strVal, data, &result)
			return result, err
		}
		return nil, fmt.Errorf("invalid list value %v", value)
	}

	for _, v := range val {
		if s.Property.Properties != nil {
			m := MapPropertyType{Property: s.Property}
			val, err := m.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result.Add(val)
		} else {
			tempProp := &Property{Type: s.Value}

			parser, err := tempProp.PropertyType()
			if err != nil {
				return nil, fmt.Errorf("invalid value type for list property type %s", s.Value)
			}
			val, err := parser.Parse(v, ctx, data)
			if err != nil {
				return nil, err
			}
			result.Add(val)
		}
	}
	return result, nil
}

func (m *MapPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {

	result := map[string]any{}

	mapVal, ok := value.(map[string]any)
	if !ok {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			err := ctx.ExecuteDecode(strVal, data, &result)
			return result, err
		}
		mapVal, ok = value.(construct.Properties)
		if !ok {
			return nil, fmt.Errorf("invalid map value %v", value)
		}
	}
	// If we are an object with sub properties then we know that we need to get the type of our sub properties to determine how we are parsed into a value
	if m.Property.Properties != nil {
		if m.Key != "" || m.Value != "" {
			return nil, fmt.Errorf("invalid map property type %s", m.Property.Name)
		}

		var errs error
		for key, prop := range m.Property.Properties {

			if _, found := mapVal[key]; !found && prop.DefaultValue != nil {
				result[key] = prop.DefaultValue
				continue
			} else if _, found := mapVal[key]; found {

				propertyType, err := m.Property.Properties[key].PropertyType()
				if err != nil {
					return nil, fmt.Errorf("unable to get property type for sub property %s: %w", key, err)
				} else if propertyType == nil {
					return nil, fmt.Errorf("%s is not a valid sub property", key)
				}
				propertyType.SetProperty(m.Property.Properties[key])
				val, err := propertyType.Parse(mapVal[key], ctx, data)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("unable to parse value for sub property %s: %w", key, err))
					continue
				}
				result[key] = val
			}
		}
		return result, nil
	}

	// Else we are a set type of map and can just loop over the values
	for key, v := range mapVal {
		keyType := m.Key
		valType := m.Value
		tempProp := &Property{Type: keyType}
		parser, err := tempProp.PropertyType()
		if err != nil {
			return nil, fmt.Errorf("invalid key type for map property type %s", keyType)
		}
		keyVal, err := parser.Parse(key, ctx, data)
		if err != nil {
			return nil, err
		}
		tempProp = &Property{Type: valType}
		parser, err = tempProp.PropertyType()
		if err != nil {
			return nil, fmt.Errorf("invalid value type for map property type %s", valType)
		}
		val, err := parser.Parse(v, ctx, data)
		if err != nil {
			return nil, err
		}
		switch keyVal := keyVal.(type) {
		case string:
			result[keyVal] = val
		case construct.ResourceId:
			result[keyVal.String()] = val
		case construct.PropertyRef:
			result[keyVal.String()] = val
		default:
			return nil, fmt.Errorf("invalid key type for map property type %s", keyType)
		}
	}
	return result, nil
}

func (a *AnyPropertyType) Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error) {
	return value, nil
}
func (s *AnyPropertyType) SetProperty(property *Property) {
}

func (m *MapPropertyType) SetProperty(property *Property) {
	m.Property = property
}

func (l *ListPropertyType) SetProperty(property *Property) {
	l.Property = property
}

func (s *SetPropertyType) SetProperty(property *Property) {
	s.Property = property
}

func (s *StringPropertyType) SetProperty(property *Property) {
}

func (i *IntPropertyType) SetProperty(property *Property) {
}

func (f *FloatPropertyType) SetProperty(property *Property) {
}

func (b *BoolPropertyType) SetProperty(property *Property) {
}

func (r *ResourcePropertyType) SetProperty(property *Property) {
}

func (p *PropertyRefPropertyType) SetProperty(property *Property) {
}

func (m *MapPropertyType) ZeroValue() any {
	return nil
}

func (l *ListPropertyType) ZeroValue() any {
	return nil
}

func (s *SetPropertyType) ZeroValue() any {
	return nil
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

func (b *AnyPropertyType) ZeroValue() any {
	return nil
}

func (r *ResourcePropertyType) ZeroValue() any {
	return construct.ResourceId{}
}

func (p *PropertyRefPropertyType) ZeroValue() any {
	return construct.PropertyRef{}
}
