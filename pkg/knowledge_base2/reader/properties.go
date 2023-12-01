package reader

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
	"gopkg.in/yaml.v3"
)

type (
	Properties map[string]*Property

	Property struct {
		Name string `json:"name" yaml:"name"`
		// Type defines the type of the property
		Type string `json:"type" yaml:"type"`

		Namespace bool `json:"namespace" yaml:"namespace"`

		DefaultValue any `json:"default_value" yaml:"default_value"`

		Required bool `json:"required" yaml:"required"`

		ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`

		DeployTime bool `json:"deploy_time" yaml:"deploy_time"`

		OperationalRule *knowledgebase.PropertyRule `json:"operational_rule" yaml:"operational_rule"`

		Properties Properties `json:"properties" yaml:"properties"`

		MinLength *int `yaml:"min_length"`
		MaxLength *int `yaml:"max_length"`

		LowerBound *float64 `yaml:"lower_bound"`
		UpperBound *float64 `yaml:"upper_bound"`

		AllowedTypes construct.ResourceList `yaml:"allowed_types"`

		SanitizeTmpl  string   `yaml:"sanitize"`
		AllowedValues []string `yaml:"allowed_values"`

		KeyProperty   knowledgebase.Property `yaml:"key_property"`
		ValueProperty knowledgebase.Property `yaml:"value_property"`

		ItemProperty knowledgebase.Property `yaml:"item_property"`

		Path string `json:"-" yaml:"-"`
	}
)

func (p *Properties) UnmarshalYAML(n *yaml.Node) error {
	type h Properties
	var p2 h
	err := n.Decode(&p2)
	if err != nil {
		return err
	}
	for name, property := range p2 {
		property.Name = name
		property.Path = name
		setChildPaths(property, name)
		p2[name] = property
	}
	*p = Properties(p2)
	return nil
}

func (p *Properties) Convert() (knowledgebase.Properties, error) {
	var errs error
	props := knowledgebase.Properties{}
	for name, prop := range *p {
		propertyType, err := prop.Convert()
		if err != nil {
			errs = fmt.Errorf("%w\n%s", errs, err.Error())
			continue
		}
		props[name] = propertyType
	}
	return props, nil
}

func (p *Property) Convert() (knowledgebase.Property, error) {
	propertyType, err := InitializeProperty(p.Type)
	if err != nil {
		return nil, err
	}

	srcVal := reflect.ValueOf(p).Elem()
	dstVal := reflect.ValueOf(propertyType).Elem()
	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		fieldName := srcVal.Type().Field(i).Name
		dstField := dstVal.FieldByName(fieldName)
		if dstField.IsValid() && dstField.CanSet() {
			// Skip nil pointers
			if (srcField.Kind() == reflect.Ptr || srcField.Kind() == reflect.Interface) && srcField.IsNil() {
				continue
			}
			// Handle sub properties so we can recurse down the tree
			if fieldName == "Properties" {
				properties, ok := srcField.Interface().(Properties)
				if !ok {
					return nil, fmt.Errorf("invalid properties")
				}
				var errs error
				props := knowledgebase.Properties{}
				for name, prop := range properties {
					propertyType, err := prop.Convert()
					if err != nil {
						errs = fmt.Errorf("%w\n%s", errs, err.Error())
						continue
					}
					props[name] = propertyType
				}
				if errs != nil {
					return nil, errs
				}
				dstField.Set(reflect.ValueOf(props))
				continue
			}

			if dstField.Type() == srcField.Type() {
				dstField.Set(srcField)
			} else {
				if conversion, found := fieldConversion[fieldName]; found {
					err := conversion(srcField, p, propertyType)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("invalid property type %s", fieldName)
				}
			}
		}
	}

	details := propertyType.Details()
	details.Path = p.Path
	return propertyType, nil
}

func setChildPaths(property *Property, currPath string) {
	for name, child := range property.Properties {
		child.Name = name
		path := currPath + "." + name
		child.Path = path
		setChildPaths(child, path)
		property.Properties[name] = child
	}
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

var fieldConversion = map[string]func(val reflect.Value, p *Property, kp knowledgebase.Property) error{
	"SanitizeTmpl": func(val reflect.Value, p *Property, kp knowledgebase.Property) error {
		sanitizeTmpl, ok := val.Interface().(string)
		if !ok {
			return fmt.Errorf("invalid sanitize template")
		}
		// generate random uuid as the name of the template
		name := uuid.New().String()
		tmpl, err := knowledgebase.NewSanitizationTmpl(name, sanitizeTmpl)
		if err != nil {
			return err
		}
		dstField := reflect.ValueOf(kp).Elem().FieldByName("SanitizeTmpl")
		dstField.Set(reflect.ValueOf(tmpl))
		return nil
	},
}

func InitializeProperty(ptype string) (knowledgebase.Property, error) {
	if ptype == "" {
		return nil, fmt.Errorf("property does not have a type")
	}
	parts := strings.Split(ptype, "(")
	p, found := initializePropertyFunc[parts[0]]
	if !found {
		return nil, fmt.Errorf("unknown property type '%s'", ptype)
	}
	var val string
	if len(parts) > 1 {
		val = strings.TrimSuffix(strings.Join(parts[1:], "("), ")")
	}
	return p(val)
}

var initializePropertyFunc = map[string]func(val string) (knowledgebase.Property, error){
	"string": func(val string) (knowledgebase.Property, error) { return &properties.StringProperty{}, nil },
	"int":    func(val string) (knowledgebase.Property, error) { return &properties.IntProperty{}, nil },
	"float":  func(val string) (knowledgebase.Property, error) { return &properties.FloatProperty{}, nil },
	"bool":   func(val string) (knowledgebase.Property, error) { return &properties.BoolProperty{}, nil },
	"resource": func(val string) (knowledgebase.Property, error) {
		id := construct.ResourceId{}
		err := id.UnmarshalText([]byte(val))
		if err != nil {
			return nil, fmt.Errorf("invalid resource id for property type %s: %w", val, err)
		}
		return &properties.ResourceProperty{
			AllowedTypes: construct.ResourceList{id},
		}, nil
	},
	"map": func(val string) (knowledgebase.Property, error) {
		if val == "" {
			return &properties.MapProperty{}, nil
		}
		args := strings.Split(val, ",")
		if len(args) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for map property type")
		}
		keyVal, err := getPropertyType(args[0])
		if err != nil {
			return nil, err
		}
		valProp, err := getPropertyType(args[1])
		if err != nil {
			return nil, err
		}
		return &properties.MapProperty{KeyProperty: keyVal, ValueProperty: valProp}, nil
	},
	"list": func(val string) (knowledgebase.Property, error) {
		if val == "" {
			return &properties.ListProperty{}, nil
		}
		itemProp, err := getPropertyType(val)
		if err != nil {
			return nil, err
		}
		return &properties.ListProperty{ItemProperty: itemProp}, nil
	},
	"set": func(val string) (knowledgebase.Property, error) {
		if val == "" {
			return &properties.SetProperty{}, nil
		}
		itemProp, err := getPropertyType(val)
		if err != nil {
			return nil, err
		}
		return &properties.SetProperty{ItemProperty: itemProp}, nil
	},
	"any": func(val string) (knowledgebase.Property, error) { return &properties.AnyProperty{}, nil },
}

func getPropertyType(ptype string) (knowledgebase.Property, error) {
	if ptype == "" {
		return nil, fmt.Errorf("property does not have a type")
	}
	parts := strings.Split(ptype, "(")
	p, found := propertyTypeMap[parts[0]]
	if !found {
		return nil, fmt.Errorf("unknown property type '%s'", ptype)
	}
	return p(), nil
}

var propertyTypeMap = map[string]func() knowledgebase.Property{
	"string":   func() knowledgebase.Property { return &properties.StringProperty{} },
	"int":      func() knowledgebase.Property { return &properties.IntProperty{} },
	"float":    func() knowledgebase.Property { return &properties.FloatProperty{} },
	"bool":     func() knowledgebase.Property { return &properties.BoolProperty{} },
	"resource": func() knowledgebase.Property { return &properties.ResourceProperty{} },
	"map":      func() knowledgebase.Property { return &properties.MapProperty{} },
	"list":     func() knowledgebase.Property { return &properties.ListProperty{} },
	"set":      func() knowledgebase.Property { return &properties.SetProperty{} },
	"any":      func() knowledgebase.Property { return &properties.AnyProperty{} },
}
