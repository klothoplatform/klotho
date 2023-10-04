package solution_context

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (ctx SolutionContext) ConfigureResource(resource *construct.Resource, configuration knowledgebase.Configuration, data knowledgebase.ConfigTemplateData, action string) error {
	if resource == nil {
		return fmt.Errorf("resource does not exist")
	}
	configCtx := knowledgebase.ConfigTemplateContext{DAG: ctx}
	var field string
	err := configCtx.ExecuteDecode(configuration.Field, data, &field)
	if err != nil {
		return err
	}

	val, err := ctx.KB.TransformToPropertyValue(resource, field, configuration.Value, configCtx, data)
	if err != nil {
		return err
	}

	switch action {
	case "set":
		err = resource.SetProperty(field, val)
		if err != nil {
			return fmt.Errorf("failed to set property %s on resource %s: %w", field, resource.ID, err)
		}
	case "add":
		err = resource.AppendProperty(field, val)
		if err != nil {
			return fmt.Errorf("failed to add property %s on resource %s: %w", field, resource.ID, err)
		}
	case "remove":
		err = resource.RemoveProperty(field, val)
		if err != nil {
			return fmt.Errorf("failed to remove property %s on resource %s: %w", field, resource.ID, err)
		}
	default:
		return fmt.Errorf("invalid action %s", action)
	}
	ctx.RecordDecision(SetPropertyDecision{
		Resource: resource.ID,
		Property: configuration.Field,
		Value:    configuration.Value,
	})
	return nil
}
