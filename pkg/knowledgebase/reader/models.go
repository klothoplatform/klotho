package reader

import (
	"fmt"
	"strings"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	Model struct {
		Name       string     `json:"name" yaml:"name"`
		Properties Properties `json:"properties" yaml:"properties"`
		Property   *Property  `json:"property" yaml:"property"`
	}
)

func (m Model) Convert() (*knowledgebase.Model, error) {
	model := &knowledgebase.Model{}
	model.Name = m.Name
	if m.Properties != nil {
		properties, err := m.Properties.Convert()
		if err != nil {
			return nil, err
		}
		model.Properties = properties
	}
	if m.Property != nil {
		property, err := m.Property.Convert()
		if err != nil {
			return nil, err
		}
		model.Property = property
	}
	return model, nil
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
