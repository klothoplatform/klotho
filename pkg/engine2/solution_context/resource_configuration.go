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
	if data.Resource != resource.ID {
		return fmt.Errorf("data resource (%s) does not match configuring resource (%s)", data.Resource, resource.ID)
	}
	field := configuration.Field

	val, err := knowledgebase.TransformToPropertyValue(
		resource.ID,
		field,
		configuration.Value,
		DynamicCtx(ctx),
		data,
	)
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

func ConstraintOperatorToAction(op constraints.ConstraintOperator) (string, error) {
	switch op {
	case constraints.AddConstraintOperator:
		return "add", nil
	case constraints.RemoveConstraintOperator:
		return "remove", nil
	case constraints.EqualsConstraintOperator:
		return "set", nil
	default:
		return "", fmt.Errorf("invalid operator %s", op)
	}
}
