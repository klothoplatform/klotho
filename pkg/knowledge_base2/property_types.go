package knowledgebase2

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	PropertyType interface {
		Parse(value any, ctx DynamicContext, data DynamicValueData) (any, error)
		SetProperty(property *Property)
		ZeroValue() any
		Contains(value any, contains any) bool
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
		if len(property.Properties) != 0 {
			return &MapPropertyType{Property: property}, nil
		}
		if len(args) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for map property type")
		}
		return &MapPropertyType{Key: args[0], Value: args[1], Property: property}, nil
	},
	"list": func(val string, p *Property) (PropertyType, error) {
		if len(p.Properties) != 0 {
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
	cloned := *p
	cloned.Properties = make(Properties, len(p.Properties))
	for k, v := range p.Properties {
		cloned.Properties[k] = v.Clone()
	}
	return &cloned
}

// ReplacePath runs a simple [strings.ReplaceAll] on the path of the property and all of its sub properties.
// NOTE: this mutates the property, so make sure to [Property.Clone] it first if you don't want that.
func (p *Property) ReplacePath(original, replacement string) {
	p.Path = strings.ReplaceAll(p.Path, original, replacement)
	for _, prop := range p.Properties {
		prop.ReplacePath(original, replacement)
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
