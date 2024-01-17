package properties

import (
	"errors"
	"fmt"
	"reflect"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	MapProperty struct {
		MinLength     *int
		MaxLength     *int
		KeyProperty   knowledgebase.Property
		ValueProperty knowledgebase.Property
		Properties    knowledgebase.Properties
		SharedPropertyFields
		knowledgebase.PropertyDetails
	}
)

func (m *MapProperty) SetProperty(resource *construct.Resource, value any) error {
	if val, ok := value.(map[string]any); ok {
		return resource.SetProperty(m.Path, val)
	}
	return fmt.Errorf("invalid resource value %v", value)
}

func (m *MapProperty) AppendProperty(resource *construct.Resource, value any) error {
	return resource.AppendProperty(m.Path, value)
}

func (m *MapProperty) RemoveProperty(resource *construct.Resource, value any) error {
	propVal, err := resource.GetProperty(m.Path)
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
		return resource.SetProperty(m.Path, propMap)
	}
	return resource.RemoveProperty(m.Path, value)
}

func (m *MapProperty) Details() *knowledgebase.PropertyDetails {
	return &m.PropertyDetails
}

func (m *MapProperty) Clone() knowledgebase.Property {
	var keyProp knowledgebase.Property
	if m.KeyProperty != nil {
		keyProp = m.KeyProperty.Clone()
	}
	var valProp knowledgebase.Property
	if m.ValueProperty != nil {
		valProp = m.ValueProperty.Clone()
	}
	var props knowledgebase.Properties
	if m.Properties != nil {
		props = m.Properties.Clone()
	}
	clone := *m
	clone.KeyProperty = keyProp
	clone.ValueProperty = valProp
	clone.Properties = props
	return &clone
}

func (m *MapProperty) GetDefaultValue(ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
	if m.DefaultValue == nil {
		return nil, nil
	}
	return m.Parse(m.DefaultValue, ctx, data)
}

func (m *MapProperty) Parse(value any, ctx knowledgebase.DynamicContext, data knowledgebase.DynamicValueData) (any, error) {
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
		case construct.ResourceId:
			result[keyVal.String()] = val
		case construct.PropertyRef:
			result[keyVal.String()] = val
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

func (m *MapProperty) Validate(resource *construct.Resource, value any, ctx knowledgebase.DynamicContext) error {
	if value == nil {
		if m.Required {
			return fmt.Errorf(knowledgebase.ErrRequiredProperty, m.Path, resource.ID)
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
	// Only validate values if its a primitive map, otherwise let the sub properties handle their own validation
	for k, v := range mapVal {
		if m.KeyProperty != nil {
			var sanitizeErr *knowledgebase.SanitizeError
			if err := m.KeyProperty.Validate(resource, k, ctx); errors.As(err, &sanitizeErr) {
				k = sanitizeErr.Sanitized.(string)
				hasSanitized = true
			} else if err != nil {
				errs = errors.Join(errs, fmt.Errorf("invalid key %v for map property type %s: %w", k, m.KeyProperty.Type(), err))
			}
		}
		if m.ValueProperty != nil {
			var sanitizeErr *knowledgebase.SanitizeError
			if err := m.ValueProperty.Validate(resource, v, ctx); errors.As(err, &sanitizeErr) {
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
		return &knowledgebase.SanitizeError{
			Input:     mapVal,
			Sanitized: validMap,
		}
	}
	return nil
}

func (m *MapProperty) SubProperties() knowledgebase.Properties {
	return m.Properties
}

func (m *MapProperty) Key() knowledgebase.Property {
	return m.KeyProperty
}

func (m *MapProperty) Value() knowledgebase.Property {
	return m.ValueProperty
}
