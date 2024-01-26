package operational_rule

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dominikbraun/graph"
	"github.com/google/uuid"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	operationalResourceAction struct {
		Step       knowledgebase.OperationalStep
		CurrentIds []construct.ResourceId
		ruleCtx    OperationalRuleContext
		numNeeded  int
	}
)

func (action *operationalResourceAction) handleOperationalResourceAction(resource *construct.Resource) error {
	if action.numNeeded == 0 {
		return nil
	}

	if action.Step.Unique && action.numNeeded > 0 {
		err := action.createUniqueResources(resource)
		if err != nil {
			return fmt.Errorf("error during operational resource action while creating unique resources: %w", err)
		}
		return nil
	}

	// we want the negative and positive case to trigger this so you can specify -1 as all available
	if action.numNeeded != 0 {
		err := action.useAvailableResources(resource)
		if err != nil {
			return fmt.Errorf("error during operational resource action while using available resources: %w", err)
		}
	}

	for action.numNeeded > 0 {
		priorityType, selector, err := action.getPriorityResourceType()
		if err != nil {
			return fmt.Errorf("cannot create resources to satisfy operational step: no resource types found for step: %w", err)
		}
		err = action.createResource(priorityType, selector, resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (action *operationalResourceAction) createUniqueResources(resource *construct.Resource) error {
	priorityType, selector, err := action.getPriorityResourceType()
	if err != nil {
		return err
	}
	// Lets check to see if the unique resource was created by some other process
	// it must be directly up/downstream and have no other dependencies in that direction
	var ids []construct.ResourceId
	if action.Step.Direction == knowledgebase.DirectionDownstream {
		ids, err = solution_context.Downstream(action.ruleCtx.Solution, resource.ID, knowledgebase.ResourceDirectLayer)
		if err != nil {
			return err
		}
	} else {
		ids, err = solution_context.Upstream(action.ruleCtx.Solution, resource.ID, knowledgebase.ResourceDirectLayer)
		if err != nil {
			return err
		}
	}
	for _, id := range ids {
		if priorityType.Matches(id) {
			var uids []construct.ResourceId
			if action.Step.Direction == knowledgebase.DirectionUpstream {
				uids, err = solution_context.Downstream(action.ruleCtx.Solution, id, knowledgebase.ResourceDirectLayer)
				if err != nil {
					return err
				}
			} else {
				uids, err = solution_context.Upstream(action.ruleCtx.Solution, id, knowledgebase.ResourceDirectLayer)
				if err != nil {
					return err
				}
			}
			if len(uids) == 1 {
				res, err := action.ruleCtx.Solution.RawView().Vertex(id)
				if err != nil {
					return err
				}
				if action.numNeeded > 0 {

					err := action.ruleCtx.addDependencyForDirection(action.Step, resource, res)
					if err != nil {
						return err
					}
					action.numNeeded--
					if action.numNeeded == 0 {
						break
					}
				}
			}
		}
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
	availableResources := make(set.Set[*construct.Resource])

	edges, err := action.ruleCtx.Solution.DataflowGraph().Edges()
	if err != nil {
		return err
	}
	resources, err := construct.TopologicalSort(action.ruleCtx.Solution.RawView())
	if err != nil {
		return err
	}

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

			for _, resId := range resources {
				res, err := action.ruleCtx.Solution.RawView().Vertex(resId)
				if err != nil {
					return err
				}
				if collectionutil.Contains(action.CurrentIds, res.ID) {
					continue
				}
				if match, err := resourceSelector.IsMatch(configCtx, action.ruleCtx.Data, res); !match {
					canUse, err := resourceSelector.CanUse(configCtx, action.ruleCtx.Data, res)
					if err != nil {
						return fmt.Errorf("error checking %s can use resource: %w", resId, err)
					}
					if !canUse {
						continue
					}
					// This can happen if an empty resource was created via path expansion, but isn't yet set up.

					tmpl, err := action.ruleCtx.Solution.KnowledgeBase().GetResourceTemplate(res.ID)
					if err != nil {
						return err
					}
					for k, v := range resourceSelector.Properties {
						v, err := knowledgebase.TransformToPropertyValue(res.ID, k, v, configCtx, action.ruleCtx.Data)
						if err != nil {
							return err
						}
						err = res.SetProperty(k, v)
						if err != nil {
							return err
						}
						if tmpl.GetProperty(k).Details().Namespace {
							oldId := res.ID
							res.ID.Namespace = resource.ID.Namespace
							err := action.ruleCtx.Solution.OperationalView().UpdateResourceID(oldId, res.ID)
							if err != nil {
								return err
							}
						}
					}
				} else if err != nil {
					return fmt.Errorf("error checking %s matches selector: %w", resId, err)
				}
				if satisfy, err := action.doesResourceSatisfyNamespace(resource, res); !satisfy {
					continue
				} else if err != nil {
					return fmt.Errorf("error checking %s satisfies namespace: %w", resId, err)
				}

				var edge construct.SimpleEdge
				if action.Step.Direction == knowledgebase.DirectionDownstream {
					edge = construct.SimpleEdge{Source: resource.ID, Target: res.ID}
				} else {
					edge = construct.SimpleEdge{Source: res.ID, Target: resource.ID}
				}
				// Check to see if the edge already exists, if it does, then we should be able to reuse the resource
				_, err = action.ruleCtx.Solution.RawView().Edge(edge.Source, edge.Target)
				if err == nil {
					availableResources.Add(res)
					continue
				} else if !errors.Is(err, graph.ErrEdgeNotFound) {
					return err
				}
				edgeTmpl := action.ruleCtx.Solution.KnowledgeBase().GetEdgeTemplate(edge.Source, edge.Target)
				if edgeTmpl == nil {
					continue
				}
				if edgeTmpl.Unique.CanAdd(edges, edge.Source, edge.Target) {
					availableResources.Add(res)
				}
			}
		}
	}
	err = action.placeResources(resource, availableResources)
	if err != nil {
		return fmt.Errorf("error during operational resource action while placing resources: %w", err)
	}
	return nil
}

func (action *operationalResourceAction) placeResources(resource *construct.Resource,
	availableResources set.Set[*construct.Resource]) error {
	placerGen, ok := placerMap[action.Step.SelectionOperator]
	if !ok {
		return fmt.Errorf("unknown selection operator %s", action.Step.SelectionOperator)
	}
	placer := placerGen()
	placer.SetCtx(action.ruleCtx)
	resources := availableResources.ToSlice()
	sort.Slice(resources, func(i, j int) bool {
		return construct.ResourceIdLess(resources[i].ID, resources[j].ID)
	})
	return placer.PlaceResources(resource, action.Step, resources, &action.numNeeded)
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
	namespaceResourceId, err := kb.GetResourcesNamespaceResource(resource)
	if err != nil {
		return false, fmt.Errorf("error during operational resource action while getting namespace resource: %w", err)
	}
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
			res, err := action.ruleCtx.Solution.RawView().Vertex(id)
			if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
				return construct.ResourceId{}, resourceSelector, err
			}
			if id.IsZero() || (res != nil && !action.Step.Unique) {
				continue
			}
			return construct.ResourceId{Provider: id.Provider, Type: id.Type, Namespace: id.Namespace, Name: id.Name}, resourceSelector, nil
		}
	}
	return construct.ResourceId{}, knowledgebase.ResourceSelector{}, fmt.Errorf("no resource types found for step, %s", action.Step.Resource)
}

func (action *operationalResourceAction) addSelectorProperties(properties map[string]any, resource *construct.Resource) error {
	template, err := action.ruleCtx.Solution.KnowledgeBase().GetResourceTemplate(resource.ID)
	if err != nil {
		return err
	}
	var errs error
	configCtx := solution_context.DynamicCtx(action.ruleCtx.Solution)
	for key, value := range properties {
		property := template.GetProperty(key)
		if property == nil {
			return fmt.Errorf("property %s not found in template %s", key, template.Id())
		}
		selectorPropertyVal, err := knowledgebase.TransformToPropertyValue(resource.ID, key, value, configCtx, action.ruleCtx.Data)
		if err != nil {
			return err
		}
		err = resource.SetProperty(key, selectorPropertyVal)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (action *operationalResourceAction) createResource(
	resourceType construct.ResourceId,
	selector knowledgebase.ResourceSelector,
	stepResource *construct.Resource,
) error {
	resId := resourceType
	if err := action.generateResourceName(&resId, stepResource.ID); err != nil {
		return err
	}
	newRes, err := knowledgebase.CreateResource(action.ruleCtx.Solution.KnowledgeBase(), resId)
	if err != nil {
		return err
	}
	if err := action.createAndAddDependency(newRes, stepResource); err != nil {
		return err
	}
	if err := action.addSelectorProperties(selector.Properties, newRes); err != nil {
		return err
	}
	return nil
}

func (action *operationalResourceAction) createAndAddDependency(res, stepResource *construct.Resource) error {
	err := action.ruleCtx.Solution.OperationalView().AddVertex(res)
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return err
	}
	err = action.ruleCtx.addDependencyForDirection(action.Step, stepResource, res)
	if err != nil {
		return err
	}
	action.numNeeded--
	return nil
}

func (action *operationalResourceAction) generateResourceName(resourceToSet *construct.ResourceId, resource construct.ResourceId) error {
	if resourceToSet.Name != "" {
		return nil
	}
	if action.Step.Unique {
		// If creating unique resources, don't need to count the total resources because the owner's name is added
		// which adds enough uniqueness against other resources in the graph. Just need to handle when the owner
		// creates multiple resources of the same type.
		suffix := ""
		if action.Step.NumNeeded > 1 {
			// If we are creating multiple resources, we want to append the number of resources we have created so far
			// so that the names are unique.
			suffix = fmt.Sprintf("-%d", action.Step.NumNeeded-action.numNeeded)
		}
		resourceToSet.Name = fmt.Sprintf("%s-%s%s", resource.Name, resourceToSet.Type, suffix)
		return nil
	}
	return generateResourceName(action.ruleCtx.Solution, resourceToSet, resource)
}

func generateResourceName(sol solution_context.SolutionContext, resourceToSet *construct.ResourceId, resource construct.ResourceId) error {
	numResources := 0
	ids, err := construct.TopologicalSort(sol.DataflowGraph())
	if err != nil {
		return err
	}
	currNames := make(set.Set[string])
	// we cannot consider things only in the namespace because when creating a resource for an operational action
	// it likely has not been namespaced yet and we dont know where it will be namespaced to
	matcher := construct.ResourceId{Provider: resourceToSet.Provider, Type: resourceToSet.Type}
	for _, id := range ids {
		if matcher.Matches(id) {
			currNames.Add(id.Name)
			numResources++
		}
	}
	// check if the current name based on the digit conflicts with an existing name and if so create a random uuid suffix
	resourceToSet.Name = fmt.Sprintf("%s-%d", resourceToSet.Type, numResources)
	if currNames.Contains(resourceToSet.Name) {
		suffix := uuid.NewString()[:8]
		resourceToSet.Name = fmt.Sprintf("%s-%s", resourceToSet.Type, suffix)
	}
	return nil
}
