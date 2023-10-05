package operational_rule

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/dominikbraun/graph"
	"github.com/iancoleman/strcase"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type (
	operationalResourceAction struct {
		Step          knowledgebase.OperationalStep
		CurrentIds    []construct.ResourceId
		PropertyEdges []construct.Edge
	}
)

func (ctx OperationalRuleContext) HandleOperationalStep(step knowledgebase.OperationalStep) (Result, error) {
	// Default to 1 resource needed
	if step.NumNeeded == 0 {
		step.NumNeeded = 1
	}

	resourceId := ctx.Data.Resource
	if resourceId.IsZero() {
		var err error
		resourceId, err = ctx.ConfigCtx.ExecuteDecodeAsResourceId(step.Resource, ctx.Data)
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
		return nil
	}

	explicitResources, _, err := step.ExtractResourcesAndTypes(ctx.ConfigCtx, ctx.Data)
	if err != nil {
		return Result{}, err
	}
	allExplicitResourcesSatisfied := true
	for _, id := range explicitResources {
		if !collectionutil.Contains(ids, id) {
			allExplicitResourcesSatisfied = false
			break
		}
	}

	if len(ids) > step.NumNeeded && allExplicitResourcesSatisfied {
		return Result{}, nil
	}

	if step.FailIfMissing {
		return Result{}, fmt.Errorf("operational resource '%s' missing when required", resource.ID)
	}

	action := operationalResourceAction{
		Step:          step,
		CurrentIds:    ids,
		PropertyEdges: edges,
	}
	return ctx.handleOperationalResourceAction(resource, action)
}

func (ctx OperationalRuleContext) handleOperationalResourceAction(
	resource *construct.Resource,
	action operationalResourceAction,
) (Result, error) {
	numNeeded := action.Step.NumNeeded - len(action.CurrentIds)
	if numNeeded <= 0 {
		// should already be checked in [HandleOperationalStep] but double check to be defensive
		return Result{}, nil
	}

	explicitResources, resourceTypes, err := action.Step.ExtractResourcesAndTypes(ctx.ConfigCtx, ctx.Data)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		AddedDependencies: action.PropertyEdges,
	}
	var errs error

	createResource := func(id construct.ResourceId) *construct.Resource {
		numNeeded--
		newRes := construct.CreateResource(id)
		err := ctx.Solution.RawView().AddVertex(newRes)
		if err != nil {
			errs = errors.Join(errs, err)
			return newRes
		}
		var edge construct.Edge
		edge, err = ctx.addDependencyForDirection(action.Step, resource, newRes)
		if err != nil {
			errs = errors.Join(errs, err)
			return newRes
		}
		result.CreatedResources = append(result.CreatedResources, newRes)
		result.AddedDependencies = append(result.AddedDependencies, edge)

		return newRes
	}

	// Add explicitly named resources first
	for _, explicitResource := range explicitResources {
		if numNeeded <= 0 {
			return Result{}, nil
		}
		_, err := ctx.Solution.RawView().Vertex(explicitResource)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			createResource(explicitResource)
		case err != nil:
			errs = errors.Join(errs, err)
			continue
		}
	}
	if errs != nil {
		return Result{}, errs
	}
	if numNeeded <= 0 {
		return result, nil
	}

	// If the rule contains classifications, we are going to get the resource types which satisfy those and put it onto the list of applicable resource types
	if len(action.Step.Classifications) > 0 {
		resourceTypes = append(resourceTypes, ctx.findResourcesWhichSatisfyStepClassifications(action.Step, resource.ID)...)
	}

	// If there are no resource types, we can't do anything since we dont understand what resources will satisfy the rule
	if len(resourceTypes) == 0 {
		return result, fmt.Errorf("no resources found that can satisfy the operational resource error")
	}

	if action.Step.Unique {
		// loop over the number of resources still needed and create them if the unique flag is true
		for numNeeded > 0 {
			typeToCreate := resourceTypes[0]
			createResource(ctx.addResourceName(typeToCreate))
		}
	}

	dynCtx := solution_context.DynamicCtx(ctx.Solution)

typeLoop:
	for _, typeToCreate := range resourceTypes {
		namespacedIds, err := ctx.Solution.KnowledgeBase().GetAllowedNamespacedResourceIds(dynCtx, typeToCreate)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		namespaceResourcesForResource := make(set.Set[construct.ResourceId])
		for _, namespacedId := range namespacedIds {
			if ctx.Solution.KnowledgeBase().HasFunctionalPath(resource.ID, namespacedId) {
				downstreams, err := solution_context.Downstream(ctx.Solution, resource.ID, knowledgebase.FirstFunctionalLayer)
				if err != nil {
					errs = errors.Join(errs, err)
					continue typeLoop
				}
				namespaceResourcesForResource.Add(downstreams...)
			}
		}

		var availableResources []*construct.Resource
		ids, err := construct.ReverseTopologicalSort(ctx.Solution.DataflowGraph())
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		resources, err := construct.ResolveIds(ctx.Solution.DataflowGraph(), ids)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, res := range resources {
			if collectionutil.Contains(action.CurrentIds, res.ID) {
				continue
			}
			if res.ID.QualifiedTypeName() == typeToCreate.QualifiedTypeName() {
				namespaceResource := ctx.Solution.KnowledgeBase().GetResourcesNamespaceResource(res)
				// needed resource is not namespaced or resource doesnt have any namespace types
				// downstream or the namespaced resource is using the right namespace
				if len(namespacedIds) == 0 ||
					len(namespaceResourcesForResource) == 0 ||
					namespaceResourcesForResource.Contains(namespaceResource) {

					availableResources = append(availableResources, res)
				}
			}
		}

		// TODO: Here we should evaluate resources based on the operator, so spread, etc so that we can order the selection of resources
		for _, res := range availableResources {
			if numNeeded <= 0 {
				break
			}
			edge, err := ctx.addDependencyForDirection(action.Step, resource, res)
			if err != nil {
				errs = errors.Join(errs, err)
				continue typeLoop
			}
			result.AddedDependencies = append(result.AddedDependencies, edge)
			numNeeded--
		}
	}
	if errs != nil {
		return Result{}, errs
	}

	for numNeeded > 0 {
		typeToCreate := resourceTypes[0]
		createResource(ctx.addResourceName(typeToCreate))
	}
	if errs != nil {
		return Result{}, errs
	}

	return result, nil
}

func (ctx OperationalRuleContext) findResourcesWhichSatisfyStepClassifications(
	step knowledgebase.OperationalStep,
	resource construct.ResourceId,
) []construct.ResourceId {
	kb := ctx.Solution.KnowledgeBase()
	// determine the type of resource necessary to satisfy the operational resource error
	var result []construct.ResourceId
	for _, res := range kb.ListResources() {
		resTempalte, err := kb.GetResourceTemplate(res.Id())
		if err != nil {
			continue
		}
		if !resTempalte.ResourceContainsClassifications(step.Classifications) {
			continue
		}
		var hasPath bool
		if step.Direction == knowledgebase.DirectionDownstream {
			hasPath = kb.HasFunctionalPath(resource, res.Id())
		} else {
			hasPath = kb.HasFunctionalPath(res.Id(), resource)
		}
		// if a type is explicilty stated as needed, we will consider it even if there isnt a direct p
		if !hasPath {
			continue
		}
		result = append(result, res.Id())
	}
	return result
}

func (ctx OperationalRuleContext) shouldReplace(step knowledgebase.OperationalStep) (bool, error) {
	if step.ReplacementCondition != "" {
		result := false
		err := ctx.ConfigCtx.ExecuteDecode(step.ReplacementCondition, ctx.Data, &result)
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
	if step.Resources != nil {
		for _, dep := range dependentResources {
			if collectionutil.Contains(step.Resources, dep.QualifiedTypeName()) {
				resourcesOfType = append(resourcesOfType, dep)
			}
		}
	} else if step.Classifications != nil {
		for _, dep := range dependentResources {
			resTemplate, err := ctx.Solution.KnowledgeBase().GetResourceTemplate(dep)
			if err != nil {
				return nil, err
			}
			if resTemplate.ResourceContainsClassifications(step.Classifications) {
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
	if err != nil {
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
	partialId.Name = strcase.ToSnake(ctx.Property.Name)
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

	// If this sets the field driving the namespace, for example,
	// then the Id could change, so replace the resource in the graph
	// to update all the edges to the new Id.
	err := construct.PropagateUpdatedId(ctx.Solution.RawView(), oldId)
	if err != nil {
		return err
	}
	return nil
}
