package operational_rule

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	OperationalResourceAction struct {
		Step       *knowledgebase.OperationalStep
		CurrentIds []construct.ResourceId
		ruleCtx    OperationalRuleContext
		numNeeded  int
	}
)

func (action *OperationalResourceAction) handleOperationalResourceAction(resource *construct.Resource) error {
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
		priorityType, err := action.getPriorityResourceType()
		if err != nil {
			return err
		}
		err = action.createResource(priorityType, resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (action *OperationalResourceAction) handleExplicitResources(resource *construct.Resource) error {
	for _, resourceSelector := range action.Step.Resources {
		if resourceSelector.Selector == "" {
			continue
		}
		decodedId, err := action.ruleCtx.ConfigCtx.ExecuteDecodeAsResourceId(resourceSelector.Selector, action.ruleCtx.Data)
		if err != nil {
			return fmt.Errorf("error during operational resource action while decoding resource selector: %w", err)
		}
		if decodedId.Name != "" {
			res, err := action.ruleCtx.Graph.GetResource(decodedId)
			if err != nil && err != graph.ErrVertexNotFound {
				return err
			}
			if res == nil {
				res = construct.CreateResource(decodedId)
				err = action.addSelectorProperties(resourceSelector.Properties, res)
				if err != nil {
					return err
				}
			}
			if resourceSelector.IsMatch(decodedId, res, action.ruleCtx.KB) {
				err = action.ruleCtx.addDependencyForDirection(action.Step, resource, res)
				if err != nil {
					return err
				}
				action.numNeeded--
			}
		}
	}
	return nil
}

func (action *OperationalResourceAction) getPriorityResourceType() (construct.ResourceId, error) {
	for _, resourceSelector := range action.Step.Resources {
		types := action.getResourceSelectorIds(resourceSelector)
		for _, id := range types {
			if id.IsZero() {
				continue
			}
			return id, nil
		}
	}
	return construct.ResourceId{}, fmt.Errorf("no resource types found for step")
}

func (action *OperationalResourceAction) createUniqueResources(resource *construct.Resource) error {
	priorityType, err := action.getPriorityResourceType()
	if err != nil {
		return err
	}
	for action.numNeeded > 0 {
		err := action.createResource(priorityType, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func (action *OperationalResourceAction) useAvailableResources(resource *construct.Resource) error {
	var availableResources []*construct.Resource
	// Next we will loop through and try to use available resources if the unique flag is not set
	for _, resourceSelector := range action.Step.Resources {
		ids := action.getResourceSelectorIds(resourceSelector)
		if len(ids) == 0 {
			continue
		}

		// because there can be multiple types if we only have classifications on the resource selector we want to loop over all ids
		for _, id := range ids {
			resources, err := action.ruleCtx.Graph.ListResources()
			if err != nil {
				return err
			}
			for _, res := range resources {
				if collectionutil.Contains(action.CurrentIds, res.ID) {
					continue
				}
				if !resourceSelector.IsMatch(id, res, action.ruleCtx.KB) {
					continue
				}
				if satisfy, err := action.doesResourceSatisfyNamespace(resource, res, id); !satisfy || err != nil {
					continue
				}
				availableResources = append(availableResources, res)
			}
		}
		err := action.placeResources(resource, availableResources)
		if err != nil {
			return fmt.Errorf("error during operational resource action while placing resources: %w", err)
		}

		if action.numNeeded <= 0 {
			return nil
		}
	}
	return nil
}

func (action *OperationalResourceAction) placeResources(resource *construct.Resource, availableResources []*construct.Resource) error {
	placerGen, ok := placerMap[action.Step.SelectionOperator]
	if !ok {
		return fmt.Errorf("unknown selection operator %s", action.Step.SelectionOperator)
	}
	placer := placerGen()
	placer.SetCtx(action.ruleCtx)
	return placer.PlaceResources(resource, *action.Step, availableResources, &action.numNeeded)
}

func (action *OperationalResourceAction) doesResourceSatisfyNamespace(stepResource *construct.Resource,
	resource *construct.Resource, typeToCreate construct.ResourceId) (bool, error) {

	namespacedIds, err := action.ruleCtx.KB.GetAllowedNamespacedResourceIds(action.ruleCtx.ConfigCtx, typeToCreate)
	if err != nil {
		return false, err
	}
	// If the type to create doesnt get namespaced, then we can ignore this satisfication
	if len(namespacedIds) == 0 {
		return true, nil
	}

	// Get all the functional resources which exist downstream of the
	var namespaceResourcesForResource []*construct.Resource
	for _, namespacedId := range namespacedIds {
		// If theres no functional path from one resource to the other, then we dont care about that namespacedId
		if action.ruleCtx.KB.HasFunctionalPath(stepResource.ID, namespacedId) {
			downstreams, err := action.ruleCtx.Graph.DownstreamOfType(stepResource, 3, namespacedId.QualifiedTypeName())
			if err != nil {
				return false, err
			}
			namespaceResourcesForResource = append(namespaceResourcesForResource, downstreams...)
		}
	}

	// If there are no functional resources downstream for the possible namespace resource types
	// we have free will to choose any of the resources available with the type of the type to create
	if len(namespaceResourcesForResource) == 0 {
		return true, nil
	}

	// for the resource we are checking if its available based on if it is namespaced
	// if it is namespaced we will ensure that it is namespaced into one of the resources downstream of the step resource
	if resource.ID.QualifiedTypeName() == typeToCreate.QualifiedTypeName() {
		namespaceResourceId := action.ruleCtx.KB.GetResourcesNamespaceResource(resource)
		var namespaceResource *construct.Resource
		if namespaceResourceId != nil {
			var err error
			namespaceResource, err = action.ruleCtx.Graph.GetResource(*namespaceResourceId)
			if err != nil {
				return false, err
			}
		}

		// needed resource is not namespaced or resource doesnt have any namespace types downstream or the namespaced resource is using the right namespace
		if collectionutil.Contains(namespaceResourcesForResource, namespaceResource) {
			return true, nil
		}
	}
	return false, nil
}

func (action *OperationalResourceAction) findResourcesWhichSatisfyStepClassifications(step knowledgebase.ResourceSelector,
	direction knowledgebase.Direction, resource *construct.Resource) []construct.ResourceId {
	// determine the type of resource necessary to satisfy the operational resource error
	var result []construct.ResourceId
	for _, res := range action.ruleCtx.KB.ListResources() {
		resTempalte, err := action.ruleCtx.KB.GetResourceTemplate(res.Id())
		if err != nil {
			continue
		}
		if !resTempalte.ResourceContainsClassifications(step.Classifications) {
			continue
		}
		var hasPath bool
		if direction == knowledgebase.Downstream {
			hasPath = action.ruleCtx.KB.HasFunctionalPath(resource.ID, res.Id())
		} else {
			hasPath = action.ruleCtx.KB.HasFunctionalPath(res.Id(), resource.ID)
		}
		// if a type is explicilty stated as needed, we will consider it even if there isnt a direct p
		if !hasPath {
			continue
		}
		result = append(result, res.Id())
	}
	return result
}

func (action *OperationalResourceAction) getResourceSelectorIds(resourceSelector knowledgebase.ResourceSelector) []construct.ResourceId {
	var ids []construct.ResourceId
	if resourceSelector.Selector != "" {
		decodedId, err := action.ruleCtx.ConfigCtx.ExecuteDecodeAsResourceId(resourceSelector.Selector, action.ruleCtx.Data)
		if err != nil {
			return ids
		}

		template, err := action.ruleCtx.KB.GetResourceTemplate(decodedId)
		if err != nil {
			return ids
		}
		if !template.ResourceContainsClassifications(resourceSelector.Classifications) {
			return ids
		}

		return []construct.ResourceId{decodedId}
	} else {
		return action.findResourcesWhichSatisfyStepClassifications(resourceSelector, knowledgebase.Downstream, nil)
	}
}

func (action *OperationalResourceAction) addSelectorProperties(properties map[string]any, resource *construct.Resource) error {
	template, err := action.ruleCtx.KB.GetResourceTemplate(resource.ID)
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

func (action *OperationalResourceAction) createResource(resourceType construct.ResourceId, stepResource *construct.Resource) error {
	newRes := construct.CreateResource(resourceType)
	action.generateResourceName(newRes, stepResource)
	err := action.ruleCtx.addDependencyForDirection(action.Step, stepResource, newRes)
	if err != nil {
		return err
	}
	action.numNeeded--
	return nil
}

func (action *OperationalResourceAction) generateResourceName(resourceToSet, resource *construct.Resource) {
	numResources := 0
	resources, err := action.ruleCtx.Graph.ListResources()
	if err != nil {
		return
	}
	for _, res := range resources {
		if res.ID.Type == resourceToSet.ID.Type {
			numResources++
		}
	}
	if action.Step.Unique {
		resourceToSet.ID.Name = fmt.Sprintf("%s-%s-%d", resourceToSet.ID.Type, resource.ID.Name, numResources)
	} else {
		resourceToSet.ID.Name = fmt.Sprintf("%s-%d", resourceToSet.ID.Type, numResources)
	}
}
