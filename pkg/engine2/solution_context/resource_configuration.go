package solution_context

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

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

// getResourcesFromPropertyReference takes a property reference and returns all the resources that are
// referenced by it. It does this by walking the property reference (split by #)
// and finding all the resources that are in the property.
func GetResourcesFromPropertyReference(
	ctx SolutionContext,
	resource construct.ResourceId,
	propertyRef string,
) (
	resources []construct.ResourceId,
	errs error,
) {
	parts := strings.Split(propertyRef, "#")
	resources = []construct.ResourceId{resource}
	if propertyRef == "" {
		return
	}
	for _, part := range parts {
		fieldValueResources := []construct.ResourceId{}
		for _, resId := range resources {
			r, err := ctx.RawView().Vertex(resId)
			if r == nil || err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to determine path satisfaction inputs. could not find resource %s: %w",
					resId, err,
				))
				continue
			}
			val, err := r.GetProperty(part)
			if err != nil || val == nil {
				continue
			}
			if id, ok := val.(construct.ResourceId); ok {
				fieldValueResources = append(fieldValueResources, id)
			} else if rval := reflect.ValueOf(val); rval.Kind() == reflect.Slice || rval.Kind() == reflect.Array {
				for i := 0; i < rval.Len(); i++ {
					idVal := rval.Index(i).Interface()
					if id, ok := idVal.(construct.ResourceId); ok {
						fieldValueResources = append(fieldValueResources, id)
					} else {
						errs = errors.Join(errs, fmt.Errorf(
							"failed to determine path satisfaction inputs. array property %s on resource %s is not a resource id",
							part, resId,
						))
					}
				}
			} else {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to determine path satisfaction inputs. property %s on resource %s is not a resource id",
					part, resId,
				))
			}
		}
		resources = fieldValueResources
	}
	return
}
