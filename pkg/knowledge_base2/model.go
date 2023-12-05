package knowledgebase2

import (
	"fmt"
)

type (
	Model struct {
		Name       string     `json:"name" yaml:"name"`
		Properties Properties `json:"properties" yaml:"properties"`
		Property   Property   `json:"property" yaml:"property"`
	}
)

// GetObjectValue returns the value of the object as the model type
func (m *Model) GetObjectValue(val any, ctx DynamicValueContext, data DynamicValueData) (any, error) {

	GetVal := func(p Property, val map[string]any) (any, error) {
		propVal, found := val[p.Details().Name]
		if !found {
			defaultVal, err := p.GetDefaultValue(ctx, data)
			if err != nil {
				return nil, err
			}
			if defaultVal != nil {
				return defaultVal, nil
			}
			return nil, fmt.Errorf("property %s not found", p.Details().Name)
		}
		return p.Parse(propVal, ctx, data)
	}
	if m.Properties != nil && m.Property != nil {
		return nil, fmt.Errorf("model has both properties and a property")
	}
	if m.Properties == nil && m.Property == nil {
		return nil, fmt.Errorf("model has neither properties nor a property")
	}

	var errs error
	if m.Properties != nil {
		obj := map[string]any{}
		for name, prop := range m.Properties {
			valMap, ok := val.(map[string]any)
			if !ok {
				errs = fmt.Errorf("%s\n%s", errs, fmt.Errorf("value for model object is not a map"))
				continue
			}
			val, err := GetVal(prop, valMap)
			if err != nil {
				errs = fmt.Errorf("%s\n%s", errs, err.Error())
				continue
			}
			obj[name] = val
		}
		return obj, errs
	} else {
		value, err := m.Property.Parse(val, ctx, data)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}
