package reader

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/properties"
	"gopkg.in/yaml.v3"
)

type (
	// Properties defines the structure of properties defined in yaml as a part of a template.
	Properties map[string]*Property

	// Property defines the structure of a property defined in yaml as a part of a template.
	// these fields must be exactly the union of all the fields in the different property types.
	Property struct {
		Name string `json:"name" yaml:"name"`
		// Type defines the type of the property
		Type string `json:"type" yaml:"type"`

		Description string `json:"description" yaml:"description"`
		IsImportant bool   `json:"important" yaml:"important"`

		Namespace bool `json:"namespace" yaml:"namespace"`

		DefaultValue any `json:"default_value" yaml:"default_value"`

		Required bool `json:"required" yaml:"required"`

		ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`

		DeployTime bool `json:"deploy_time" yaml:"deploy_time"`

		OperationalRule *knowledgebase.PropertyRule `json:"operational_rule" yaml:"operational_rule"`

		Properties Properties `json:"properties" yaml:"properties"`

		// MinLength defines the minimum length of a string, list, set, or map (number of entries)
		MinLength *int `yaml:"min_length"`
		MaxLength *int `yaml:"max_length"`

		MinValue *float64 `yaml:"min_value"`
		MaxValue *float64 `yaml:"max_value"`

		// UniqueItems defines whether the items in a list or set must be unique
		UniqueItems *bool `yaml:"unique_items"`
		// UniqueKeys defines whether the keys in a map must be unique (default true)
		UniqueKeys *bool `yaml:"unique_keys"`

		AllowedTypes construct.ResourceList `yaml:"allowed_types"`

		SanitizeTmpl  string   `yaml:"sanitize"`
		AllowedValues []string `yaml:"allowed_values"`

		KeyProperty   *Property `yaml:"key_property"`
		ValueProperty *Property `yaml:"value_property"`

		ItemProperty *Property `yaml:"item_property"`

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
	return props, errs
}

func (p *Property) Convert() (knowledgebase.Property, error) {
	propertyType, err := InitializeProperty(p.Type)
	if err != nil {
		return nil, err
	}
	propertyType.Details().Path = p.Path

	srcVal := reflect.ValueOf(p).Elem()
	dstVal := reflect.ValueOf(propertyType).Elem()
	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		fieldName := srcVal.Type().Field(i).Name
		dstField := dstVal.FieldByName(fieldName)
		if !dstField.IsValid() || !dstField.CanSet() {
			continue
		}
		// Skip nil pointers
		if (srcField.Kind() == reflect.Ptr || srcField.Kind() == reflect.Interface) && srcField.IsNil() {
			continue
			// skip empty arrays and slices
		} else if (srcField.Kind() == reflect.Array || srcField.Kind() == reflect.Slice) && srcField.Len() == 0 {
			continue
		}
		// Handle sub properties so we can recurse down the tree
		switch fieldName {
		case "Properties":
			properties := srcField.Interface().(Properties)
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
				return nil, fmt.Errorf("could not convert sub properties: %w", errs)
			}
			dstField.Set(reflect.ValueOf(props))
			continue

		case "KeyProperty", "ValueProperty":
			if !strings.HasPrefix(p.Type, "map") {
				return nil, fmt.Errorf("property must be 'map' (was %s) for %s", p.Type, fieldName)
			}
			keyType, valueType, hasElementTypes := strings.Cut(
				strings.TrimSuffix(strings.TrimPrefix(p.Type, "map("), ")"),
				",",
			)
			elemProp := srcField.Interface().(*Property)
			// Add the element's type if it is not specified but is on the parent.
			// For example, 'map(string,string)' on the parent means the key_property doesn't need 'type: string'
			if hasElementTypes {
				if fieldName == "KeyProperty" {
					if elemProp.Type != "" && elemProp.Type != keyType {
						return nil, fmt.Errorf("key property type must be %s (was %s)", keyType, elemProp.Type)
					} else if elemProp.Type == "" {
						elemProp.Type = keyType
					}
				} else {
					if elemProp.Type != "" && elemProp.Type != valueType {
						return nil, fmt.Errorf("value property type must be %s (was %s)", valueType, elemProp.Type)
					} else if elemProp.Type == "" {
						elemProp.Type = valueType
					}
				}
			}
			converted, err := elemProp.Convert()
			if err != nil {
				return nil, fmt.Errorf("could not convert %s: %w", fieldName, err)
			}
			srcField = reflect.ValueOf(converted)
		case "ItemProperty":
			if !strings.HasPrefix(p.Type, "list") && !strings.HasPrefix(p.Type, "set") {
				return nil, fmt.Errorf("property must be 'list' or 'set' (was %s) for %s", p.Type, fieldName)
			}
			hasItemType := strings.Contains(p.Type, "(")
			elemProp := srcField.Interface().(*Property)
			if hasItemType {
				itemType := strings.TrimSuffix(
					strings.TrimPrefix(strings.TrimPrefix(p.Type, "list("), "set("),
					")",
				)
				if elemProp.Type != "" && elemProp.Type != itemType {
					return nil, fmt.Errorf("item property type must be %s (was %s)", itemType, elemProp.Type)
				} else if elemProp.Type == "" {
					elemProp.Type = itemType
				}
			}
			converted, err := elemProp.Convert()
			if err != nil {
				return nil, fmt.Errorf("could not convert %s: %w", fieldName, err)
			}
			srcField = reflect.ValueOf(converted)
		}

		if srcField.Type().AssignableTo(dstField.Type()) {
			dstField.Set(srcField)
			continue
		}

		if dstField.Kind() == reflect.Ptr && srcField.Kind() == reflect.Ptr {
			if srcField.Type().Elem().AssignableTo(dstField.Type().Elem()) {
				dstField.Set(srcField)
				continue
			} else if srcField.Type().Elem().ConvertibleTo(dstField.Type().Elem()) {
				val := srcField.Elem().Convert(dstField.Type().Elem())
				// set dest field to a pointer of val
				dstField.Set(reflect.New(dstField.Type().Elem()))
				dstField.Elem().Set(val)
				continue
			}
		}

		if conversion, found := fieldConversion[fieldName]; found {
			err := conversion(srcField, p, propertyType)
			if err != nil {
				return nil, err
			}
			continue
		}

		return nil, fmt.Errorf(
			"could not assign %s#%s (%s) to field in %T (%s)",
			p.Path, fieldName, srcField.Type(), propertyType, dstField.Type(),
		)

	}

	return propertyType, nil
}

func setChildPaths(property *Property, currPath string) {
	for name, child := range property.Properties {
		child.Name = name
		path := currPath + "." + name
		child.Path = path
		setChildPaths(child, path)
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

// fieldConversion is a map providing functionality on how to convert inputs into our internal types if they are not inherently the same structure
var fieldConversion = map[string]func(val reflect.Value, p *Property, kp knowledgebase.Property) error{
	"SanitizeTmpl": func(val reflect.Value, p *Property, kp knowledgebase.Property) error {
		sanitizeTmpl, ok := val.Interface().(string)
		if !ok {
			return fmt.Errorf("invalid sanitize template")
		}
		if sanitizeTmpl == "" {
			return nil
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

var initializePropertyFunc map[string]func(val string) (knowledgebase.Property, error)

func init() {
	// initializePropertyFunc initialization is deferred to prevent cyclic initialization (a compiler error) with `InitializeProperty`
	initializePropertyFunc = map[string]func(val string) (knowledgebase.Property, error){
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
				return nil, fmt.Errorf("invalid number of arguments for map property type: %s", val)
			}
			keyVal, err := InitializeProperty(args[0])
			if err != nil {
				return nil, err
			}
			valProp, err := InitializeProperty(args[1])
			if err != nil {
				return nil, err
			}
			return &properties.MapProperty{KeyProperty: keyVal, ValueProperty: valProp}, nil
		},
		"list": func(val string) (knowledgebase.Property, error) {
			if val == "" {
				return &properties.ListProperty{}, nil
			}
			itemProp, err := InitializeProperty(val)
			if err != nil {
				return nil, err
			}
			return &properties.ListProperty{ItemProperty: itemProp}, nil
		},
		"set": func(val string) (knowledgebase.Property, error) {
			if val == "" {
				return &properties.SetProperty{}, nil
			}
			itemProp, err := InitializeProperty(val)
			if err != nil {
				return nil, err
			}
			return &properties.SetProperty{ItemProperty: itemProp}, nil
		},
		"any": func(val string) (knowledgebase.Property, error) { return &properties.AnyProperty{}, nil },
	}
}
