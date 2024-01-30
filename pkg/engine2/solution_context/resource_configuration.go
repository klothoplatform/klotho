package solution_context

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

//go:generate mockgen -source=./resource_configuration.go --destination=../operational_eval/resource_configurer_mock_test.go --package=operational_eval

type (
	ResourceConfigurer interface {
		ConfigureResource(
			resource *construct.Resource,
			configuration knowledgebase.Configuration,
			data knowledgebase.DynamicValueData,
			action string,
			userInitiated bool,
		) error
	}

	Configurer struct {
		Ctx SolutionContext
	}
)

func (c *Configurer) ConfigureResource(
	resource *construct.Resource,
	configuration knowledgebase.Configuration,
	data knowledgebase.DynamicValueData,
	action string,
	userInitiated bool,
) error {
	if resource == nil {
		return fmt.Errorf("resource does not exist")
	}
	if resource.Imported && !userInitiated {
		return fmt.Errorf("cannot configure imported resource %s", resource.ID)
	}
	if data.Resource != resource.ID {
		return fmt.Errorf("data resource (%s) does not match configuring resource (%s)", data.Resource, resource.ID)
	}
	field := configuration.Field
	rt, err := c.Ctx.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil {
		return err
	}
	property := rt.GetProperty(field)
	if property == nil {
		return fmt.Errorf("failed to get property %s on resource %s: %w", field, resource.ID, err)
	}
	val, err := knowledgebase.TransformToPropertyValue(
		resource.ID,
		field,
		configuration.Value,
		DynamicCtx(c.Ctx),
		data,
	)
	if err != nil {
		return err
	}

	switch action {
	case "set":
		err = property.SetProperty(resource, val)
		if err != nil {
			return fmt.Errorf("failed to set property %s on resource %s: %w", field, resource.ID, err)
		}
		err = AddDeploymentDependenciesFromVal(c.Ctx, resource, val)
		if err != nil {
			return fmt.Errorf("failed to add deployment dependencies from property %s on resource %s: %w", field, resource.ID, err)
		}
	case "add":
		err = property.AppendProperty(resource, val)
		if err != nil {
			return fmt.Errorf("failed to add property %s on resource %s: %w", field, resource.ID, err)
		}
		err = AddDeploymentDependenciesFromVal(c.Ctx, resource, val)
		if err != nil {
			return fmt.Errorf("failed to add deployment dependencies from property %s on resource %s: %w", field, resource.ID, err)
		}
	case "remove":
		err = property.RemoveProperty(resource, val)
		if err != nil {
			return fmt.Errorf("failed to remove property %s on resource %s: %w", field, resource.ID, err)
		}
	default:
		return fmt.Errorf("invalid action %s", action)
	}
	c.Ctx.RecordDecision(SetPropertyDecision{
		Resource: resource.ID,
		Property: configuration.Field,
		Value:    configuration.Value,
	})
	return nil
}

func AddDeploymentDependenciesFromVal(
	ctx SolutionContext,
	resource *construct.Resource,
	val any,
) error {
	var errs error
	ids := getResourcesFromValue(val)
	for _, id := range ids {
		if resource.ID.Matches(id) {
			continue
		}
		err := ctx.DeploymentGraph().AddEdge(resource.ID, id)
		if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
			errs = errors.Join(errs, fmt.Errorf("failed to add deployment dependency from %s to %s: %w", resource.ID, id, err))
		}
	}
	return errs
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

func getResourcesFromValue(val any) (ids []construct.ResourceId) {
	if val == nil {
		return
	}
	switch v := val.(type) {
	case construct.ResourceId:
		ids = []construct.ResourceId{v}
	case construct.PropertyRef:
		ids = []construct.ResourceId{v.Resource}
	default:
		rval := reflect.ValueOf(val)
		switch rval.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < reflect.ValueOf(val).Len(); i++ {
				idVal := rval.Index(i).Interface()
				ids = append(ids, getResourcesFromValue(idVal)...)
			}
		case reflect.Map:
			for _, key := range reflect.ValueOf(val).MapKeys() {
				idVal := rval.MapIndex(key).Interface()
				ids = append(ids, getResourcesFromValue(idVal)...)
			}
		}
	}
	return
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
					"failed to get resources from property reference. could not find resource %s: %w",
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
							"failed to get resources from property reference. array property %s on resource %s is not a resource id",
							part, resId,
						))
					}
				}
			} else {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to get resources from property reference. property %s on resource %s is not a resource id",
					part, resId,
				))
			}
		}
		resources = fieldValueResources
	}
	return
}
