package knowledgebase2

import (
	"fmt"
	"reflect"
)

type (
	Model struct {
		Name       string     `json:"name" yaml:"name"`
		Properties Properties `json:"properties" yaml:"properties"`
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
				delete(properties, name)
				for name, prop := range model.Properties {
					// since properties are pointers and models can be reused, we need to clone the property from the model itself
					newProp := prop.Clone()
					newProp.Path = fmt.Sprintf("%s.%s", name, prop.Path)
					properties[name] = newProp
				}
				if property != nil {
					updateModelPaths(property)
				}
			} else {
				p.Properties = models[*modelType].Properties.Clone()
				modelString := fmt.Sprintf("model(%s)", *modelType)
				if p.Type == modelString {
					p.Type = "map"
				} else if p.Type == fmt.Sprintf("list(%s)", modelString) {
					p.Type = "list"
				}
				updateModelPaths(p)
			}
		}
		err := updateModels(p, p.Properties, models)
		if err != nil {
			return err
		}
		// Substitute default values
		if p.DefaultValue != nil {
			if prop, found := properties[name]; found {
				prop.DefaultValue = p.DefaultValue
			} else {
				// If we are spreading a model into the resource template we need to inspect
				// the keys of the default values for field names
				rval := reflect.ValueOf(p.DefaultValue)
				for _, key := range rval.MapKeys() {
					if !rval.MapIndex(key).IsValid() || rval.MapIndex(key).IsNil() {
						continue
					}
					keyStr, found := key.Interface().(string)
					if !found {
						return fmt.Errorf("map key is not a string")
					}
					if prop, found := properties[keyStr]; found {
						prop.DefaultValue = rval.MapIndex(key).Interface()
					}
				}
			}
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
