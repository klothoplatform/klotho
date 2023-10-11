package operational_rule

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	operationalResourceAction struct {
		Step       knowledgebase.OperationalStep
		CurrentIds []construct.ResourceId
		ruleCtx    OperationalRuleContext
		numNeeded  int
		result     Result
	}
)

func (action *operationalResourceAction) handleOperationalResourceAction(resource *construct.Resource) error {

	if action.numNeeded <= 0 {
		return nil
	}

	// First we will loop through and attempt to create all explicit resources
	err := action.handleExplicitResources(resource)
	if err != nil {
		return fmt.Errorf("error during operational resource action while handling explicit resources: %w", err)
	}

	if action.Step.Unique && action.numNeeded > 0 {
		err = action.createUniqueResources(resource)
		if err != nil {
			return fmt.Errorf("error during operational resource action while creating unique resources: %w", err)
		}
		return nil
	}

	if action.numNeeded > 0 {
		err = action.useAvailableResources(resource)
		if err != nil {
			return fmt.Errorf("error during operational resource action while using available resources: %w", err)
		}
	}

	for action.numNeeded > 0 {
		priorityType, selector, err := action.getPriorityResourceType()
		if err != nil {
			return err
		}
		err = action.createResource(priorityType, selector, resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (action *operationalResourceAction) handleExplicitResources(resource *construct.Resource) error {
	configCtx := solution_context.DynamicCtx(action.ruleCtx.Solution)
	for _, resourceSelector := range action.Step.Resources {
		if resourceSelector.Selector == "" {
			continue
		}
		decodedId, err := configCtx.ExecuteDecodeAsResourceId(resourceSelector.Selector, action.ruleCtx.Data)
		if err != nil {
			return fmt.Errorf("error during operational resource action while decoding resource selector: %w", err)
		}
		if decodedId.Name != "" {
			res, err := action.ruleCtx.Solution.RawView().Vertex(decodedId)
			if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
				return err
			}
			if res == nil {
				res = construct.CreateResource(decodedId)
				err = action.addSelectorProperties(resourceSelector.Properties, res)
				if err != nil {
					return err
				}
			}
			if resourceSelector.IsMatch(configCtx, action.ruleCtx.Data, res) {
				err := action.createAndAddDependency(res, resource)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (action *operationalResourceAction) createUniqueResources(resource *construct.Resource) error {
	priorityType, selector, err := action.getPriorityResourceType()
	if err != nil {
		return err
	}
	for action.numNeeded > 0 {
		err := action.createResource(priorityType, selector, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (action *operationalResourceAction) useAvailableResources(resource *construct.Resource) error {
	configCtx := solution_context.DynamicCtx(action.ruleCtx.Solution)
	var availableResources []*construct.Resource
	// Next we will loop through and try to use available resources if the unique flag is not set
	for _, resourceSelector := range action.Step.Resources {
		ids, err := resourceSelector.ExtractResourceIds(configCtx, action.ruleCtx.Data)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			continue
		}

		// because there can be multiple types if we only have classifications on the resource selector we want to loop over all ids
		for _, id := range ids {

			// if there is no functional path for the id then we can skip it since we know its not available to satisfy a valid graph
			if action.Step.Direction == knowledgebase.DirectionDownstream &&
				!action.ruleCtx.Solution.KnowledgeBase().HasFunctionalPath(resource.ID, id) {
				continue
			} else if action.Step.Direction == knowledgebase.DirectionUpstream &&
				!action.ruleCtx.Solution.KnowledgeBase().HasFunctionalPath(id, resource.ID) {
				continue
			}

			resources, err := construct.ToplogicalSort(action.ruleCtx.Solution.RawView())
			if err != nil {
				return err
			}
			for _, resId := range resources {
				res, err := action.ruleCtx.Solution.RawView().Vertex(resId)
				if err != nil {
					return err
				}
				if collectionutil.Contains(action.CurrentIds, res.ID) {
					continue
				}
				if !resourceSelector.IsMatch(configCtx, action.ruleCtx.Data, res) {
					continue
				}
				if satisfy, err := action.doesResourceSatisfyNamespace(resource, res); !satisfy || err != nil {
					continue
				}
				availableResources = append(availableResources, res)
			}
		}
	}
	result, err := action.placeResources(resource, availableResources)
	action.result.Append(result)
	if err != nil {
		return fmt.Errorf("error during operational resource action while placing resources: %w", err)
	}
	return nil
}

func (action *operationalResourceAction) placeResources(resource *construct.Resource,
	availableResources []*construct.Resource) (Result, error) {
	placerGen, ok := placerMap[action.Step.SelectionOperator]
	if !ok {
		return Result{}, fmt.Errorf("unknown selection operator %s", action.Step.SelectionOperator)
	}
	placer := placerGen()
	placer.SetCtx(action.ruleCtx)
	return placer.PlaceResources(resource, action.Step, availableResources, &action.numNeeded)
}

func (action *operationalResourceAction) doesResourceSatisfyNamespace(stepResource *construct.Resource, resource *construct.Resource) (bool, error) {
	kb := action.ruleCtx.Solution.KnowledgeBase()
	namespacedIds, err := kb.GetAllowedNamespacedResourceIds(solution_context.DynamicCtx(action.ruleCtx.Solution), resource.ID)
	if err != nil {
		return false, err
	}
	// If the type to create doesnt get namespaced, then we can ignore this satisfication
	if len(namespacedIds) == 0 {
		return true, nil
	}

	// Get all the functional resources which exist downstream of the step resource
	var namespaceResourcesForResource []construct.ResourceId
	for _, namespacedId := range namespacedIds {
		// If theres no functional path from one resource to the other, then we dont care about that namespacedId
		if kb.HasFunctionalPath(stepResource.ID, namespacedId) {
			downstreams, err := solution_context.Downstream(action.ruleCtx.Solution, stepResource.ID, knowledgebase.FirstFunctionalLayer)
			if err != nil {
				return false, err
			}
			for _, downstream := range downstreams {
				if namespacedId.Matches(downstream) {
					namespaceResourcesForResource = append(namespaceResourcesForResource, downstream)
				}
			}
		}
	}

	// If there are no functional resources downstream for the possible namespace resource types
	// we have free will to choose any of the resources available with the type of the type to create
	if len(namespaceResourcesForResource) == 0 {
		return true, nil
	}

	// for the resource we are checking if its available based on if it is namespaced
	// if it is namespaced we will ensure that it is namespaced into one of the resources downstream of the step resource
	namespaceResourceId := kb.GetResourcesNamespaceResource(resource)
	var namespaceResource *construct.Resource
	if !namespaceResourceId.IsZero() {
		var err error
		namespaceResource, err = action.ruleCtx.Solution.RawView().Vertex(namespaceResourceId)
		if err != nil {
			return false, err
		}

		// needed resource is not namespaced or resource doesnt have any namespace types downstream or the namespaced resource is using the right namespace
		if !collectionutil.Contains(namespaceResourcesForResource, namespaceResource.ID) {
			return false, nil
		}
	}
	return true, nil
}

func (action *operationalResourceAction) getPriorityResourceType() (
	construct.ResourceId,
	knowledgebase.ResourceSelector,
	error,
) {
	for _, resourceSelector := range action.Step.Resources {
		ids, err := resourceSelector.ExtractResourceIds(solution_context.DynamicCtx(action.ruleCtx.Solution), action.ruleCtx.Data)
		if err != nil {
			return construct.ResourceId{}, resourceSelector, err
		}
		for _, id := range ids {
			if id.IsZero() {
				continue
			}
			return construct.ResourceId{Provider: id.Provider, Type: id.Type}, resourceSelector, nil
		}
	}
	return construct.ResourceId{}, knowledgebase.ResourceSelector{}, fmt.Errorf("no resource types found for step")
}

func (action *operationalResourceAction) addSelectorProperties(properties map[string]any, resource *construct.Resource) error {
	template, err := action.ruleCtx.Solution.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil {
		return err
	}
	for key, value := range properties {
		property := template.GetProperty(key)
		if property == nil {
			return fmt.Errorf("property %s not found in template %s", key, template.Id())
		}
		err := resource.SetProperty(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (action *operationalResourceAction) createResource(
	resourceType construct.ResourceId,
	selector knowledgebase.ResourceSelector,
	stepResource *construct.Resource,
) error {
	newRes := construct.CreateResource(resourceType)
	action.addSelectorProperties(selector.Properties, newRes)
	action.generateResourceName(&newRes.ID, stepResource.ID)
	action.createAndAddDependency(newRes, stepResource)
	return nil
}

func (action *operationalResourceAction) createAndAddDependency(res, stepResource *construct.Resource) error {
	err := action.ruleCtx.Solution.RawView().AddVertex(res)
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return err
	}
	if err == nil {
		action.result.CreatedResources = append(action.result.CreatedResources, res)
	}
	var edge construct.Edge
	edge, err = action.ruleCtx.addDependencyForDirection(action.Step, stepResource, res)
	if err != nil {
		return err
	}
	action.result.AddedDependencies = append(action.result.AddedDependencies, edge)
	action.numNeeded--
	return nil
}

func (action *operationalResourceAction) generateResourceName(resourceToSet *construct.ResourceId, resource construct.ResourceId) {
	numResources := 0
	ids, err := construct.ToplogicalSort(action.ruleCtx.Solution.DataflowGraph())
	if err != nil {
		return
	}
	for _, id := range ids {
		if id.Type == resourceToSet.Type {
			numResources++
		}
	}
	if action.Step.Unique {
		resourceToSet.Name = fmt.Sprintf("%s-%s-%d", resourceToSet.Type, resource.Name, numResources)
	} else {
		resourceToSet.Name = fmt.Sprintf("%s-%d", resourceToSet.Type, numResources)
	}
}
