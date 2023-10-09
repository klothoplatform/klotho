package solution_context

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func ConfigureResource(
	ctx SolutionContext,
	resource *construct.Resource,
	configuration knowledgebase.Configuration,
	data knowledgebase.DynamicValueData,
	action string,
) error {
	if resource == nil {
		return fmt.Errorf("resource does not exist")
	}
	configCtx := knowledgebase.DynamicValueContext{DAG: ctx.DataflowGraph(), KB: ctx.KnowledgeBase()}
	var field string
	err := configCtx.ExecuteDecode(configuration.Field, data, &field)
	if err != nil {
		return err
	}

	if configuration.Value == nil {
		err = resource.RemoveProperty(field, nil)
		if err != nil {
			return fmt.Errorf("failed to remove property (due to empty value) %s on resource %s: %w", field, resource.ID, err)
		}
		return nil
	}

	val, err := knowledgebase.TransformToPropertyValue(resource, field, configuration.Value, configCtx, data)
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

func ApplyConfigureConstraint(ctx SolutionContext, res *construct.Resource, rc constraints.ResourceConstraint) error {
	ctx = ctx.With("constraint", rc)
	configuration := knowledgebase.Configuration{
		Field: rc.Property,
		Value: rc.Value,
	}
	switch rc.Operator {
	case constraints.AddConstraintOperator:
		return ConfigureResource(ctx, res, configuration, knowledgebase.DynamicValueData{Resource: res.ID}, "add")
	case constraints.RemoveConstraintOperator:
		return ConfigureResource(ctx, res, configuration, knowledgebase.DynamicValueData{Resource: res.ID}, "remove")
	case constraints.EqualsConstraintOperator:
		return ConfigureResource(ctx, res, configuration, knowledgebase.DynamicValueData{Resource: res.ID}, "set")
	default:
		return fmt.Errorf("invalid operator %s", rc.Operator)
	}
}
