package knowledgebase2

import (
	"fmt"
)

type (
	Model struct {
		Name       string     `json:"name" yaml:"name"`
		Properties Properties `json:"properties" yaml:"properties"`
	}
)

func updateModels(properties Properties, models map[string]*Model) error {
	for name, p := range properties {
		modelType := p.ModelType()
		if modelType != nil {
			if p.Properties != nil {
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
					properties[name] = prop
				}
			} else {
				p.Properties = models[*modelType].Properties
				modelString := fmt.Sprintf("model(%s)", *modelType)
				if p.Type == modelString {
					p.Type = "map"
				} else if p.Type == fmt.Sprintf("list(%s)", modelString) {
					p.Type = "list"
				}
			}
		} else {
			updateModels(p.Properties, models)
		}
	}
	return nil
}
