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
	val, err := ctx.kb.TransformToPropertyValue(resource, configuration.Field, configuration.Value, configCtx, data)
	if err != nil {
		return err
	}

	switch action {
	case "set":

		resource.SetProperty(configuration.Field, val)
	case "add":
		resource.AppendProperty(configuration.Field, val)
	case "remove":
		resource.RemoveProperty(configuration.Field, val)
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
