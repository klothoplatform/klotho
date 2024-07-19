package properties

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
)

type (
	MapProperty struct {
		MinLength     *int
		MaxLength     *int
		KeyProperty   property.Property
		ValueProperty property.Property
		Properties    property.PropertyMap
		SharedPropertyFields
		property.PropertyDetails
	}
)

func (m *MapProperty) SetProperty(properties construct.Properties, value any) error {
	if val, ok := value.(map[string]any); ok {
		return properties.SetProperty(m.Path, val)
	}
	return fmt.Errorf("invalid properties value %v", value)
}

func (m *MapProperty) AppendProperty(properties construct.Properties, value any) error {
	return properties.AppendProperty(m.Path, value)
}

func (m *MapProperty) RemoveProperty(properties construct.Properties, value any) error {
	propVal, err := properties.GetProperty(m.Path)
	if errors.Is(err, construct.ErrPropertyDoesNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if propVal == nil {
		return nil
	}
	propMap, ok := propVal.(map[string]any)
	if !ok {
		return fmt.Errorf("error attempting to remove map property: invalid property value %v", propVal)
	}
	if val, ok := value.(map[string]any); ok {
		for k, v := range val {
			if val, found := propMap[k]; found && reflect.DeepEqual(val, v) {
				delete(propMap, k)
			}
		}
		return properties.SetProperty(m.Path, propMap)
	}
	return properties.RemoveProperty(m.Path, value)
}

func (m *MapProperty) Details() *property.PropertyDetails {
	return &m.PropertyDetails
}

func (m *MapProperty) Clone() property.Property {
	var keyProp property.Property
	if m.KeyProperty != nil {
		keyProp = m.KeyProperty.Clone()
	}
	var valProp property.Property
	if m.ValueProperty != nil {
		valProp = m.ValueProperty.Clone()
	}
	var props property.PropertyMap
	if m.Properties != nil {
		props = m.Properties.Clone()
	}
	clone := *m
	clone.KeyProperty = keyProp
	clone.ValueProperty = valProp
	clone.Properties = props
	return &clone
}

func (m *MapProperty) GetDefaultValue(ctx property.ExecutionContext, data any) (any, error) {
	if m.DefaultValue == nil {
		return nil, nil
	}
	return m.Parse(m.DefaultValue, ctx, data)
}

func (m *MapProperty) Parse(value any, ctx property.ExecutionContext, data any) (any, error) {
	result := map[string]any{}

	mapVal, ok := value.(map[string]any)
	if !ok {
		// before we fail, check to see if the entire value is a template
		if strVal, ok := value.(string); ok {
			err := ctx.ExecuteUnmarshal(strVal, data, &result)
			return result, err
		}
		mapVal, ok = value.(construct.Properties)
		if !ok {
			return nil, fmt.Errorf("invalid map value %v", value)
		}
	}
	// If we are an object with sub properties then we know that we need to get the type of our sub properties to determine how we are parsed into a value
	if len(m.Properties) != 0 {
		var errs error
		for key, prop := range m.Properties {
			if _, found := mapVal[key]; found {
				val, err := prop.Parse(mapVal[key], ctx, data)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("unable to parse value for sub property %s: %w", key, err))
					continue
				}
				result[key] = val
			} else {
				val, err := prop.GetDefaultValue(ctx, data)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("unable to get default value for sub property %s: %w", key, err))
					continue
				}
				if val == nil {
					continue
				}
				result[key] = val
			}
		}
	}

	if m.KeyProperty == nil || m.ValueProperty == nil {
		return result, nil
	}

	// Else we are a set type of map and can just loop over the values
	for key, v := range mapVal {
		keyVal, err := m.KeyProperty.Parse(key, ctx, data)
		if err != nil {
			return nil, err
		}
		val, err := m.ValueProperty.Parse(v, ctx, data)
		if err != nil {
			return nil, err
		}
		switch keyVal := keyVal.(type) {
		case string:
			result[keyVal] = val
		//case constructs.ConstructId:
		//	result[keyVal.String()] = val
		//case construct.PropertyRef:
		//	result[keyVal.String()] = val
		default:
			return nil, fmt.Errorf("invalid key type for map property type %s", keyVal)
		}
	}
	return result, nil
}

func (m *MapProperty) ZeroValue() any {
	return nil
}

func (m *MapProperty) Contains(value any, contains any) bool {
	mapVal, ok := value.(map[string]any)
	if !ok {
		return false
	}
	containsMap, ok := contains.(map[string]any)
	if !ok {
		return false
	}
	for k, v := range containsMap {
		if val, found := mapVal[k]; found || reflect.DeepEqual(val, v) {
			return true
		}
	}
	for _, v := range mapVal {
		for _, cv := range containsMap {
			if reflect.DeepEqual(v, cv) {
				return true
			}
		}
	}
	return false
}

func (m *MapProperty) Type() string {
	if m.KeyProperty != nil && m.ValueProperty != nil {
		return fmt.Sprintf("map(%s,%s)", m.KeyProperty.Type(), m.ValueProperty.Type())
	}
	return "map"
}

func (m *MapProperty) Validate(properties construct.Properties, value any) error {
	if value == nil {
		if m.Required {
			return fmt.Errorf(property.ErrRequiredProperty, m.Path)
		}
		return nil
	}
	mapVal, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid map value %v", value)
	}
	if m.MinLength != nil {
		if len(mapVal) < *m.MinLength {
			return fmt.Errorf("map value %v is too short. min length is %d", value, *m.MinLength)
		}
	}
	if m.MaxLength != nil {
		if len(mapVal) > *m.MaxLength {
			return fmt.Errorf("map value %v is too long. max length is %d", value, *m.MaxLength)
		}
	}
	var errs error
	hasSanitized := false
	validMap := make(map[string]any)
	// Only validate values if it's a primitive map, otherwise let the sub properties handle their own validation
	for k, v := range mapVal {
		if m.KeyProperty != nil {
			var sanitizeErr *property.SanitizeError
			if err := m.KeyProperty.Validate(properties, k); errors.As(err, &sanitizeErr) {
				k = sanitizeErr.Sanitized.(string)
				hasSanitized = true
			} else if err != nil {
				errs = errors.Join(errs, fmt.Errorf("invalid key %v for map property type %s: %w", k, m.KeyProperty.Type(), err))
			}
		}
		if m.ValueProperty != nil {
			var sanitizeErr *property.SanitizeError
			if err := m.ValueProperty.Validate(properties, v); errors.As(err, &sanitizeErr) {
				v = sanitizeErr.Sanitized
				hasSanitized = true
			} else if err != nil {
				errs = errors.Join(errs, fmt.Errorf("invalid value %v for map property type %s: %w", v, m.ValueProperty.Type(), err))
			}
		}
		validMap[k] = v
	}
	if errs != nil {
		return errs
	}
	if hasSanitized {
		return &property.SanitizeError{
			Input:     mapVal,
			Sanitized: validMap,
		}
	}
	return nil
}

func (m *MapProperty) SubProperties() property.PropertyMap {
	return m.Properties
}

func (m *MapProperty) Key() property.Property {
	return m.KeyProperty
}

func (m *MapProperty) Value() property.Property {
	return m.ValueProperty
}
