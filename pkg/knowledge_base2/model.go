package knowledgebase2

import (
	"fmt"
)

type (
	Model struct {
		Name       string     `json:"name" yaml:"name"`
		Properties Properties `json:"properties" yaml:"properties"`
		Property   *Property  `json:"property" yaml:"property"`
	}
)

func updateModels(property *Property, properties Properties, models map[string]*Model) error {
	for name, p := range properties {
		modelType := p.ModelType()
		if modelType != nil {
			if len(p.Properties) != 0 {
				return fmt.Errorf("property %s has properties but is labeled as a model", name)
			}
			model := models[*modelType]
			if model == nil || model.Properties == nil {
				return fmt.Errorf("model %s not found", *modelType)
			}
			// We know that this means we want the properties to be spread onto the resource
			if p.Name == *modelType {
				if model.Property != nil {
					return fmt.Errorf("model %s as property can not be spread into properties", *modelType)
				}
				delete(properties, name)
				for name, prop := range model.Properties {
					// since properties are pointers and models can be reused, we need to clone the property from the model itself
					newProp := prop.Clone()
					newProp.Path = fmt.Sprintf("%s.%s", name, prop.Path)

					// we also need to check if the current property has a default and propagate it lower
					if p.DefaultValue != nil {
						defaultMap, ok := p.DefaultValue.(map[string]any)
						if !ok {
							return fmt.Errorf("default value for %s is not a map", p.Path)
						}
						newProp.DefaultValue = defaultMap[name]
					}
					properties[name] = newProp
				}
				if property != nil {
					if err := updateModelPaths(property); err != nil {
						return err
					}
				}
			} else {
				m := models[*modelType]
				if m.Properties != nil {
					p.Properties = models[*modelType].Properties.Clone()
					modelString := fmt.Sprintf("model(%s)", *modelType)
					if p.Type == modelString {
						p.Type = "map"
					} else if p.Type == fmt.Sprintf("list(%s)", modelString) {
						p.Type = "list"
					}
					if err := updateModelPaths(p); err != nil {
						return err
					}
				} else if m.Property != nil {
					p = m.Property.Clone()
				}
			}
		}
		err := updateModels(p, p.Properties, models)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateModelPaths(p *Property) error {
	for _, prop := range p.Properties {
		prop.Path = fmt.Sprintf("%s.%s", p.Path, prop.Name)
		err := updateModelPaths(prop)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetObjectValue returns the value of the object as the model type
func (m *Model) GetObjectValue(val any, ctx DynamicValueContext, data DynamicValueData) (any, error) {

	GetVal := func(p *Property, val map[string]any) (any, error) {
		pType, err := p.PropertyType()
		if err != nil {
			return nil, err
		}
		propVal, found := val[p.Name]
		if !found {
			if p.DefaultValue != nil {
				return p.DefaultValue, nil
			}
			return nil, fmt.Errorf("property %s not found", p.Name)
		}

		return pType.Parse(propVal, ctx, data)
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
		pType, err := m.Property.PropertyType()
		if err != nil {
			return nil, err
		}
		value, err := pType.Parse(val, ctx, data)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
}
