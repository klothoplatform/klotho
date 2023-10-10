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

func (ctx OperationalRuleContext) HandleOperationalStep(step knowledgebase.OperationalStep) (Result, error) {
	// Default to 1 resource needed
	if step.NumNeeded == 0 {
		step.NumNeeded = 1
	}

	dyn := solution_context.DynamicCtx(ctx.Solution)

	resourceId := ctx.Data.Resource
	if resourceId.IsZero() {
		var err error
		resourceId, err = dyn.ExecuteDecodeAsResourceId(step.Resource, ctx.Data)
		if err != nil {
			return Result{}, err
		}
	}
	resource, err := ctx.Solution.RawView().Vertex(resourceId)
	if err != nil {
		return Result{}, fmt.Errorf("resource %s not found: %w", resourceId, err)
	}

	replace, err := ctx.shouldReplace(step)
	if err != nil {
		return Result{}, err
	}

	// If we are replacing we want to remove all dependencies and clear the property
	// otherwise we want to add dependencies from the property and gather the resources which satisfy the step
	var ids []construct.ResourceId
	var edges []construct.Edge
	if ctx.Property != nil {
		if replace {
			err := ctx.clearProperty(step, resource, ctx.Property.Path)
			if err != nil {
				return Result{}, err
			}
		}
		var err error
		ids, edges, err = ctx.addDependenciesFromProperty(step, resource, ctx.Property.Path)
		if err != nil {
			return Result{}, err
		}
	} else { // an edge rule won't have a Property
		ids, err = ctx.getResourcesForStep(step, resource.ID)
		if err != nil {
			return Result{}, err
		}
		if replace {
			for _, id := range ids {
				err := ctx.Solution.RawView().RemoveEdge(id, resource.ID)
				if err != nil {
					return Result{}, err
				}
			}
		}
		ids = []construct.ResourceId{}
	}

	if len(ids) >= step.NumNeeded {
		return Result{}, nil
	}

	if step.FailIfMissing {
		return Result{}, fmt.Errorf("operational resource '%s' missing when required", resource.ID)
	}

	action := operationalResourceAction{
		Step:       step,
		CurrentIds: ids,
		result: Result{
			AddedDependencies: edges,
		},
		numNeeded: step.NumNeeded - len(ids),
		ruleCtx:   ctx,
	}
	err = action.handleOperationalResourceAction(resource)
	if err != nil {
		return Result{}, err
	}
	return action.result, nil
}

func (ctx OperationalRuleContext) shouldReplace(step knowledgebase.OperationalStep) (bool, error) {
	if step.ReplacementCondition != "" {
		result := false
		dyn := solution_context.DynamicCtx(ctx.Solution)
		err := dyn.ExecuteDecode(step.ReplacementCondition, ctx.Data, &result)
		if err != nil {
			return result, err
		}
		return result, nil
	}
	return false, nil
}

func (ctx OperationalRuleContext) getResourcesForStep(step knowledgebase.OperationalStep, resource construct.ResourceId) ([]construct.ResourceId, error) {
	var dependentResources []construct.ResourceId
	var resourcesOfType []construct.ResourceId
	var err error
	if step.Direction == knowledgebase.DirectionUpstream {
		dependentResources, err = solution_context.Upstream(ctx.Solution, resource, knowledgebase.FirstFunctionalLayer)
		if err != nil {
			return nil, err
		}
	} else {
		dependentResources, err = solution_context.Downstream(ctx.Solution, resource, knowledgebase.FirstFunctionalLayer)
		if err != nil {
			return nil, err
		}
	}
	for _, dep := range dependentResources {

		dependentRes, err := ctx.Solution.RawView().Vertex(dep)
		if err != nil {
			return nil, err
		}
		for _, resourceSelector := range step.Resources {
			if resourceSelector.IsMatch(solution_context.DynamicCtx(ctx.Solution), ctx.Data, dependentRes) {
				resourcesOfType = append(resourcesOfType, dep)

			}
		}
	}
	return resourcesOfType, nil
}

func (ctx OperationalRuleContext) addDependenciesFromProperty(
	step knowledgebase.OperationalStep,
	resource *construct.Resource,
	propertyName string,
) ([]construct.ResourceId, []construct.Edge, error) {
	val, err := resource.GetProperty(propertyName)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting property %s on resource %s: %w", propertyName, resource.ID, err)
	}

	var edges []construct.Edge

	addDep := func(id construct.ResourceId) error {
		dep, err := ctx.Solution.RawView().Vertex(id)
		if err != nil {
			return err
		}
		edge, err := ctx.addDependencyForDirection(step, resource, dep)
		edges = append(edges, edge)
		return err
	}

	switch val := val.(type) {
	case construct.ResourceId:
		return []construct.ResourceId{val}, edges, addDep(val)

	case []construct.ResourceId:
		var errs error
		for _, id := range val {
			errs = errors.Join(errs, addDep(id))
		}
		return val, edges, errs

	case []any:
		var errs error
		var ids []construct.ResourceId
		for _, elem := range val {
			if id, ok := elem.(construct.ResourceId); ok {
				ids = append(ids, id)
				errs = errors.Join(errs, addDep(id))
			}
		}
		return ids, edges, errs
	}
	return nil, nil, nil
}

func (ctx OperationalRuleContext) clearProperty(step knowledgebase.OperationalStep, resource *construct.Resource, propertyName string) error {
	val, err := resource.GetProperty(propertyName)
	if err != nil {
		return err
	}

	switch val := val.(type) {
	case construct.ResourceId:
		err := ctx.removeDependencyForDirection(step.Direction, resource.ID, val)
		if err != nil {
			return err
		}
		return resource.RemoveProperty(propertyName, nil)

	case []construct.ResourceId:
		var errs error
		for _, id := range val {
			errs = errors.Join(errs, ctx.removeDependencyForDirection(step.Direction, resource.ID, id))
		}
		if errs != nil {
			return errs
		}
		return resource.RemoveProperty(propertyName, nil)

	case []any:
		var errs error
		for _, elem := range val {
			if id, ok := elem.(construct.ResourceId); ok {
				errs = errors.Join(errs, ctx.removeDependencyForDirection(step.Direction, resource.ID, id))
			}
		}
		if errs != nil {
			return errs
		}
		return resource.RemoveProperty(propertyName, nil)
	}
	return nil
}

func (ctx OperationalRuleContext) addDependencyForDirection(
	step knowledgebase.OperationalStep,
	resource, dependentResource *construct.Resource,
) (construct.Edge, error) {
	var edge construct.Edge
	if step.Direction == knowledgebase.DirectionUpstream {
		edge = construct.Edge{Source: dependentResource.ID, Target: resource.ID}
	} else {
		edge = construct.Edge{Source: resource.ID, Target: dependentResource.ID}
	}
	err := ctx.Solution.RawView().AddEdge(edge.Source, edge.Target)
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		return edge, err
	}
	return edge, ctx.setField(resource, dependentResource, step)
}

func (ctx OperationalRuleContext) removeDependencyForDirection(direction knowledgebase.Direction, resource, dependentResource construct.ResourceId) error {
	if direction == knowledgebase.DirectionUpstream {
		return ctx.Solution.RawView().RemoveEdge(dependentResource, resource)
	} else {
		return ctx.Solution.RawView().RemoveEdge(resource, dependentResource)
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
		if currResId, ok := res.(construct.ResourceId); ok {
			if res != nil && res != fieldResource.ID {
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
		zap.S().Infof("set field %s#%s to %s", resource.ID, ctx.Property.Path, fieldResource.ID)
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

	// If this sets the field driving the namespace, for example,
	// then the Id could change, so replace the resource in the graph
	// to update all the edges to the new Id.
	err := construct.PropagateUpdatedId(ctx.Solution.RawView(), oldId)
	if err != nil {
		return err
	}
	return nil
}
