package operational_rule

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dominikbraun/graph"
	"github.com/iancoleman/strcase"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

func (ctx OperationalRuleContext) HandleOperationalStep(step knowledgebase.OperationalStep) error {
	// Default to 1 resource needed
	if step.NumNeeded == 0 {
		step.NumNeeded = 1
	}

	dyn := solution_context.DynamicCtx(ctx.Solution)

	resourceId := ctx.Data.Resource
	if resourceId.IsZero() {
		var err error
		resourceId, err = knowledgebase.ExecuteDecodeAsResourceId(dyn, step.Resource, ctx.Data)
		if err != nil {
			return err
		}
	}
	resource, err := ctx.Solution.OperationalView().Vertex(resourceId)
	if err != nil {
		return fmt.Errorf("resource %s not found: %w", resourceId, err)
	}

	// If we are replacing we want to remove all dependencies and clear the property
	// otherwise we want to add dependencies from the property and gather the resources which satisfy the step
	var ids []construct.ResourceId
	if ctx.Property != nil {
		var err error
		ids, err = ctx.addDependenciesFromProperty(step, resource, ctx.Property.Path)
		if err != nil {
			return err
		}
	} else { // an edge rule won't have a Property
		ids, err = ctx.getResourcesForStep(step, resource.ID)
		if err != nil {
			return err
		}
	}

	if len(ids) >= step.NumNeeded {
		return nil
	}

	if step.FailIfMissing {
		return fmt.Errorf("operational resource '%s' missing when required", resource.ID)
	}

	action := operationalResourceAction{
		Step:       step,
		CurrentIds: ids,
		numNeeded:  step.NumNeeded - len(ids),
		ruleCtx:    ctx,
	}
	return action.handleOperationalResourceAction(resource)
}

func (ctx OperationalRuleContext) getResourcesForStep(step knowledgebase.OperationalStep, resource construct.ResourceId) ([]construct.ResourceId, error) {
	var ids []construct.ResourceId
	var err error
	if step.Direction == knowledgebase.DirectionUpstream {
		ids, err = solution_context.Upstream(ctx.Solution, resource, knowledgebase.FirstFunctionalLayer)
	} else {
		ids, err = solution_context.Downstream(ctx.Solution, resource, knowledgebase.FirstFunctionalLayer)
	}
	if err != nil {
		return nil, err
	}

	resources, err := construct.ResolveIds(ctx.Solution.RawView(), ids)
	if err != nil {
		return nil, fmt.Errorf("could not resolve ids for 'getResourcesForStep': %w", err)
	}
	dyn := solution_context.DynamicCtx(ctx.Solution)

	var resourcesOfType []construct.ResourceId
	for _, dep := range resources {
		for _, resourceSelector := range step.Resources {
			if match, err := resourceSelector.IsMatch(dyn, ctx.Data, dep); match {
				resourcesOfType = append(resourcesOfType, dep.ID)
			} else if err != nil {
				return nil, fmt.Errorf("error checking if %s is side effect of %s: %w", dep.ID, resource, err)
			}
		}
	}
	return resourcesOfType, nil
}

func (ctx OperationalRuleContext) addDependenciesFromProperty(
	step knowledgebase.OperationalStep,
	resource *construct.Resource,
	propertyName string,
) ([]construct.ResourceId, error) {
	val, err := resource.GetProperty(propertyName)
	if err != nil {
		return nil, fmt.Errorf("error getting property %s on resource %s: %w", propertyName, resource.ID, err)
	}
	if val == nil {
		return nil, nil
	}

	addDep := func(id construct.ResourceId) error {
		dep, err := ctx.Solution.RawView().Vertex(id)
		if err != nil {
			return fmt.Errorf("could not add dep to %s from %s#%s: %w", id, resource.ID, propertyName, err)
		}
		if _, err := ctx.Solution.RawView().Edge(resource.ID, dep.ID); err == nil {
			return nil
		}
		err = ctx.addDependencyForDirection(step, resource, dep)
		if err != nil {
			return err
		}
		return nil
	}

	switch val := val.(type) {
	case construct.ResourceId:
		if val.IsZero() {
			return nil, nil
		}
		return []construct.ResourceId{val}, addDep(val)

	case []construct.ResourceId:
		var errs error
		for _, id := range val {
			errs = errors.Join(errs, addDep(id))
		}
		return val, errs

	case []any:
		var errs error
		var ids []construct.ResourceId
		for _, elem := range val {
			if id, ok := elem.(construct.ResourceId); ok {
				ids = append(ids, id)
				errs = errors.Join(errs, addDep(id))
			}
		}
		return ids, errs
	}
	return nil, fmt.Errorf("cannot add dependencies from property %s on resource %s", propertyName, resource.ID)
}

func (ctx OperationalRuleContext) clearProperty(step knowledgebase.OperationalStep, resource *construct.Resource, propertyName string) error {
	val, err := resource.GetProperty(propertyName)
	if err != nil {
		return err
	}
	if val == nil {
		return nil
	}

	kb := ctx.Solution.KnowledgeBase()

	removeDep := func(id construct.ResourceId) error {
		err := ctx.removeDependencyForDirection(step.Direction, resource.ID, id)
		if err != nil {
			return err
		}
		if kb.GetFunctionality(id) == knowledgebase.Unknown {
			return reconciler.RemoveResource(ctx.Solution, id, false)
		}
		return nil
	}

	switch val := val.(type) {
	case construct.ResourceId:
		err := removeDep(val)
		if err != nil {
			return err
		}
		return resource.RemoveProperty(propertyName, nil)

	case []construct.ResourceId:
		var errs error
		for _, id := range val {
			errs = errors.Join(errs, removeDep(id))
		}
		if errs != nil {
			return errs
		}
		return resource.RemoveProperty(propertyName, nil)

	case []any:
		var errs error
		for _, elem := range val {
			if id, ok := elem.(construct.ResourceId); ok {
				errs = errors.Join(errs, removeDep(id))
			}
		}
		if errs != nil {
			return errs
		}
		return resource.RemoveProperty(propertyName, nil)
	}
	return fmt.Errorf("cannot clear property %s on resource %s", propertyName, resource.ID)
}

func (ctx OperationalRuleContext) addDependencyForDirection(
	step knowledgebase.OperationalStep,
	resource, dependentResource *construct.Resource,
) error {
	var edge construct.Edge
	if step.Direction == knowledgebase.DirectionUpstream {
		edge = construct.Edge{Source: dependentResource.ID, Target: resource.ID}
	} else {
		edge = construct.Edge{Source: resource.ID, Target: dependentResource.ID}
	}
	err := ctx.Solution.OperationalView().AddEdge(edge.Source, edge.Target)
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		return err
	}
	return ctx.setField(resource, dependentResource, step)
}

func (ctx OperationalRuleContext) removeDependencyForDirection(direction knowledgebase.Direction, resource, dependentResource construct.ResourceId) error {
	if direction == knowledgebase.DirectionUpstream {
		return ctx.Solution.OperationalView().RemoveEdge(dependentResource, resource)
	} else {
		return ctx.Solution.OperationalView().RemoveEdge(resource, dependentResource)
	}
}

func (ctx OperationalRuleContext) addResourceName(partialId construct.ResourceId) construct.ResourceId {
	// TODO handle cases when multiple resources want to use the same ID, such as `aws:subnet:myvpc:public` by adding an
	// incrementing number to them.
	if ctx.Property != nil {
		partialId.Name = strcase.ToSnake(ctx.Property.Name)
		return partialId
	}
	partialId.Name = fmt.Sprintf("%s_%s", ctx.Data.Edge.Source.Name, ctx.Data.Edge.Target.Name)
	return partialId
}

func (ctx OperationalRuleContext) setField(resource, fieldResource *construct.Resource, step knowledgebase.OperationalStep) error {
	if ctx.Property == nil {
		return nil
	}
	// snapshot the ID from before any field changes
	oldId := resource.ID

	if ctx.Property.IsPropertyTypeScalar() {
		res, err := resource.GetProperty(ctx.Property.Path)
		if err != nil {
			zap.S().Debugf("property %s not found on resource %s", ctx.Property.Path, resource.ID)
		}
		// If the current field is a resource id we will compare it against the one passed in to see if we need to remove the current resource
		if currResId, ok := res.(construct.ResourceId); ok && !currResId.IsZero() {
			if res != fieldResource.ID {
				err = ctx.removeDependencyForDirection(step.Direction, resource.ID, currResId)
				if err != nil {
					return err
				}
				zap.S().Infof("Removing old field value for '%s' (%s) for %s", ctx.Property.Path, res, fieldResource.ID)
				// Remove the old field value if it's unused
				err = reconciler.RemoveResource(ctx.Solution, currResId, false)
				if err != nil {
					return err
				}
			}
		}
		// Right now we only enforce the top level properties if they have rules, so we can assume the path is equal to the name of the property
		err = resource.SetProperty(ctx.Property.Path, fieldResource.ID)
		if err != nil {
			return fmt.Errorf("error setting field %s#%s with %s: %w", resource.ID, ctx.Property.Path, fieldResource.ID, err)
		}
		zap.S().Debugf("set field %s#%s to %s", resource.ID, ctx.Property.Path, fieldResource.ID)
		// See if we need to namespace the resource due to setting the property
		if ctx.Property.Namespace {
			resource.ID.Namespace = fieldResource.ID.Name
		}
	} else {
		// First lets check if the array already contains the id. if it does we dont want to append it
		res, err := resource.GetProperty(ctx.Property.Path)
		if err != nil {
			zap.S().Debugf("property %s not found on resource %s", ctx.Property.Path, resource.ID)
		}
		resVal := reflect.ValueOf(res)
		if resVal.IsValid() && (resVal.Kind() == reflect.Slice || resVal.Kind() == reflect.Array) {
			// If the current field is a resource id we will compare it against the one passed in to see if we need to remove the current resource
			for i := 0; i < resVal.Len(); i++ {
				currResId, ok := resVal.Index(i).Interface().(construct.ResourceId)
				if !ok {
					continue
				}
				if !currResId.IsZero() && currResId == fieldResource.ID {
					return nil
				}
			}
		}
		// Right now we only enforce the top level properties if they have rules, so we can assume the path is equal to the name of the property
		err = resource.AppendProperty(ctx.Property.Path, []construct.ResourceId{fieldResource.ID})
		if err != nil {
			return fmt.Errorf("error appending field %s#%s with %s: %w", resource.ID, ctx.Property.Path, fieldResource.ID, err)
		}
		zap.S().Infof("appended field %s#%s with %s", resource.ID, ctx.Property.Path, fieldResource.ID)
	}

	if ctx.Data.Resource.Matches(oldId) {
		ctx.Data.Resource = resource.ID
	}

	if ctx.Data.Edge != nil {
		if ctx.Data.Edge.Source.Matches(oldId) {
			ctx.Data.Edge.Source = resource.ID
		}
		if ctx.Data.Edge.Target.Matches(oldId) {
			ctx.Data.Edge.Target = resource.ID
		}
	}
	return nil
}
