package inputs

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/properties"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"gopkg.in/yaml.v3"
)

type (
	// InputTemplateMap defines the structure of properties defined in YAML as a part of a template.
	InputTemplateMap map[string]*InputTemplate

	// InputTemplate defines the structure of a property defined in YAML as a part of a template.
	// these fields must be exactly the union of all the fields in the different property types.
	InputTemplate struct {
		Name string `json:"name" yaml:"name"`
		// Type defines the type of the property
		Type string `json:"type" yaml:"type"`
		// Description defines the description of the property
		Description string `json:"description" yaml:"description"`
		// DefaultValue defines the default value of the property
		DefaultValue any `json:"default_value" yaml:"default_value"`
		// Required defines whether the property is required
		Required bool `json:"required" yaml:"required"`

		// Properties defines the sub properties of a key_value_list, map, list, or set
		Properties InputTemplateMap `json:"properties" yaml:"properties"`

		// MinLength defines the minimum length of a string, list, set, or map (number of entries)
		MinLength *int `yaml:"min_length"`
		// MaxLength defines the maximum length of a string, list, set, or map (number of entries)
		MaxLength *int `yaml:"max_length"`

		// MinValue defines the minimum value of an int or float
		MinValue *float64 `yaml:"min_value"`
		// MaxValue defines the maximum value of an int or float
		MaxValue *float64 `yaml:"max_value"`

		// UniqueItems defines whether the items in a list or set must be unique
		UniqueItems *bool `yaml:"unique_items"`
		// UniqueKeys defines whether the keys in a map must be unique (default true)
		UniqueKeys *bool `yaml:"unique_keys"`
		// SanitizeTmpl is a go template to sanitize user input when setting the property
		SanitizeTmpl string `yaml:"sanitize"`
		// AllowedValues defines an enumeration of allowed values for a string, int, float, or bool
		AllowedValues []string `yaml:"allowed_values"`

		// KeyProperty is the property of the keys in a key_value_list or map
		KeyProperty *InputTemplate `yaml:"key_property"`
		// ValueProperty is the property of the values in a key_value_list or map
		ValueProperty *InputTemplate `yaml:"value_property"`

		// ItemProperty is the property of the items in a list or set
		ItemProperty *InputTemplate `yaml:"item_property"`

		// Path is the path to the property in the template
		// this field is derived and is	not part of the yaml
		Path string `json:"-" yaml:"-"`
	}

	PropertyType       string
	FieldConverterFunc func(val reflect.Value, p *InputTemplate, kp property.Property) error
)

var (
	StringPropertyType       PropertyType = "string"
	IntPropertyType          PropertyType = "int"
	FloatPropertyType        PropertyType = "float"
	BoolPropertyType         PropertyType = "bool"
	MapPropertyType          PropertyType = "map"
	ListPropertyType         PropertyType = "list"
	SetPropertyType          PropertyType = "set"
	AnyPropertyType          PropertyType = "any"
	PathPropertyType         PropertyType = "path"
	KeyValueListPropertyType PropertyType = "key_value_list"
	ConstructPropertyType    PropertyType = "construct"
)

func (p *InputTemplateMap) UnmarshalYAML(n *yaml.Node) error {
	type h InputTemplateMap
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
	*p = InputTemplateMap(p2)
	return nil
}

func (p *InputTemplateMap) Convert() (property.PropertyMap, error) {
	var errs error
	props := property.PropertyMap{}
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

func (p *InputTemplate) Convert() (property.Property, error) {
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
			propMap := srcField.Interface().(InputTemplateMap)
			var errs error
			props := property.PropertyMap{}
			for name, prop := range propMap {
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
			switch {
			case strings.HasPrefix(p.Type, "map"):
				keyType, valueType, hasElementTypes := strings.Cut(
					strings.TrimSuffix(strings.TrimPrefix(p.Type, "map("), ")"),
					",",
				)
				elemProp := srcField.Interface().(*InputTemplate)
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
			case strings.HasPrefix(p.Type, "key_value_list"):
				keyType, valueType, hasElementTypes := strings.Cut(
					strings.TrimSuffix(strings.TrimPrefix(p.Type, "key_value_list("), ")"),
					",",
				)
				keyType = strings.TrimSpace(keyType)
				valueType = strings.TrimSpace(valueType)
				elemProp := srcField.Interface().(*InputTemplate)
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
			default:
				return nil, fmt.Errorf("property must be 'map' or 'key_value_list' (was %s) for %s", p.Type, fieldName)
			}
		case "ItemProperty":
			hasItemType := strings.Contains(p.Type, "(")
			elemProp := srcField.Interface().(*InputTemplate)
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

func setChildPaths(property *InputTemplate, currPath string) {
	for name, child := range property.Properties {
		child.Name = name
		path := currPath + "." + name
		child.Path = path
		setChildPaths(child, path)
	}
}

func (p InputTemplateMap) Clone() InputTemplateMap {
	newProps := make(InputTemplateMap, len(p))
	for k, v := range p {
		newProps[k] = v.Clone()
	}
	return newProps
}

func (p *InputTemplate) Clone() *InputTemplate {
	cloned := *p
	cloned.Properties = make(InputTemplateMap, len(p.Properties))
	for k, v := range p.Properties {
		cloned.Properties[k] = v.Clone()
	}
	return &cloned
}

// fieldConversion is a map providing functionality on how to convert inputs into our internal types if they are not inherently the same structure
var fieldConversion = map[string]FieldConverterFunc{
	"SanitizeTmpl": func(val reflect.Value, p *InputTemplate, kp property.Property) error {
		sanitizeTmpl, ok := val.Interface().(string)
		if !ok {
			return fmt.Errorf("invalid sanitize template")
		}
		if sanitizeTmpl == "" {
			return nil
		}
		tmpl, err := property.NewSanitizationTmpl(kp.Details().Name, sanitizeTmpl)
		if err != nil {
			return err
		}
		dstField := reflect.ValueOf(kp).Elem().FieldByName("SanitizeTmpl")
		dstField.Set(reflect.ValueOf(tmpl))
		return nil
	},
}

func InitializeProperty(ptype string) (property.Property, error) {
	if ptype == "" {
		return nil, fmt.Errorf("property does not have a type")
	}
	baseType, typeArgs, err := GetTypeInfo(ptype)
	if err != nil {
		return nil, err
	}
	switch baseType {
	case MapPropertyType:
		if len(typeArgs) == 0 {
			return &properties.MapProperty{}, nil
		}
		if len(typeArgs) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for map property type: %s", ptype)
		}
		keyVal, err := InitializeProperty(typeArgs[0])
		if err != nil {
			return nil, err
		}
		valProp, err := InitializeProperty(typeArgs[1])
		if err != nil {
			return nil, err
		}
		return &properties.MapProperty{KeyProperty: keyVal, ValueProperty: valProp}, nil
	case ListPropertyType:
		if len(typeArgs) == 0 {
			return &properties.ListProperty{}, nil
		}
		if len(typeArgs) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for list property type: %s", ptype)
		}
		itemProp, err := InitializeProperty(typeArgs[0])
		if err != nil {
			return nil, err
		}
		return &properties.ListProperty{ItemProperty: itemProp}, nil
	case SetPropertyType:
		if len(typeArgs) == 0 {
			return &properties.SetProperty{}, nil
		}
		if len(typeArgs) != 1 {
			return nil, fmt.Errorf("invalid number of arguments for set property type: %s", ptype)
		}
		itemProp, err := InitializeProperty(typeArgs[0])
		if err != nil {
			return nil, err
		}
		return &properties.SetProperty{ItemProperty: itemProp}, nil
	case KeyValueListPropertyType:
		if len(typeArgs) == 0 {
			return &properties.KeyValueListProperty{}, nil
		}
		if len(typeArgs) != 2 {
			return nil, fmt.Errorf("invalid number of arguments for %s property type: %s", KeyValueListPropertyType, ptype)
		}
		keyPropType := typeArgs[0]
		valPropType := typeArgs[1]
		keyProp, err := InitializeProperty(keyPropType)
		keyProp.Details().Name = "Key"
		if err != nil {
			return nil, err
		}
		valProp, err := InitializeProperty(valPropType)
		valProp.Details().Name = "Value"
		if err != nil {
			return nil, err
		}
		return &properties.KeyValueListProperty{KeyProperty: keyProp, ValueProperty: valProp}, nil
	case ConstructPropertyType:
		var allowedTypes []property.ConstructType
		if len(typeArgs) > 0 {
			for _, t := range typeArgs {
				var id property.ConstructType
				err := id.FromString(t)
				if err != nil {
					return nil, fmt.Errorf("invalid construct type %s: %w", t, err)
				}
				allowedTypes = append(allowedTypes, id)
			}
		}
		return &properties.ConstructProperty{AllowedTypes: allowedTypes}, nil
	case AnyPropertyType:
		return &properties.AnyProperty{}, nil
	case StringPropertyType:
		return &properties.StringProperty{}, nil
	case IntPropertyType:
		return &properties.IntProperty{}, nil
	case FloatPropertyType:
		return &properties.FloatProperty{}, nil
	case BoolPropertyType:
		return &properties.BoolProperty{}, nil
	case PathPropertyType:
		return &properties.PathProperty{}, nil
	default:
		return nil, fmt.Errorf("unknown property type '%s'", baseType)
	}

}

var funcRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?$`)
var argRegex = regexp.MustCompile(`[^,]+`)

func GetTypeInfo(t string) (propType PropertyType, args []string, err error) {
	matches := funcRegex.FindStringSubmatch(t)
	if matches == nil {
		return "", nil, fmt.Errorf("invalid property type %s", t)
	}
	propType = PropertyType(matches[1])
	args = argRegex.FindAllString(matches[2], -1)
	for i, arg := range args {
		args[i] = strings.TrimSpace(arg)
	}

	return propType, args, nil
}
