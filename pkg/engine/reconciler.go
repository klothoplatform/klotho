package engine

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"go.uber.org/zap"
)

// ignoreCriteria determines if we can delete a resource because the knowledge base in use by the engine, shows that the initial resource is dependent on the sub resource for deletion.
// If the sub resource is deletion dependent on any of the dependent resources passed in then we will determine weather we can delete the dependent resource first.
func (e *Engine) ignoreCriteria(resource core.Resource, dependentResources []core.Resource) bool {
DEP:
	for _, dep := range dependentResources {
		found := false
		for _, res := range e.Context.EndState.GetDownstreamResources(resource) {
			if dep == res {
				found = true
				det, _ := e.KnowledgeBase.GetEdge(resource, dep)
				if !det.DeletetionDependent {
					return false
				}
				continue DEP
			}
		}
		for _, res := range e.Context.EndState.GetUpstreamResources(resource) {
			if dep == res {
				found = true
				det, _ := e.KnowledgeBase.GetEdge(dep, resource)
				if !det.DeletetionDependent {
					return false
				}
				continue DEP
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// delete resource is used by the engine to remove resources from the resource graph in its context.
//
// if explicit is set it is meant to show that a user has explicitly requested for the resource to be deleted or that the resource requested is being deleted by its parent resource
// if overrideExplicit is set, it means that the explicit delete request still has to satisfy the resources delete criteria. If it is set to false, then the explicit deletion request is always performed
func (e *Engine) deleteResource(resource core.Resource, explicit bool, overrideExplicit bool) bool {
	log := zap.S().With(zap.String("id", resource.Id().String()))
	log.Debug("Deleting resource")
	graph := e.Context.EndState
	deletionCriteria := resource.DeleteContext()
	upstreamNodes := e.KnowledgeBase.GetTrueUpstream(resource, graph)
	downstreamNodes := e.KnowledgeBase.GetTrueDownstream(resource, graph)
	if deletionCriteria.RequiresExplicitDelete && !explicit {
		log.Debug("Cannot delete resource as it was not explicitly requested")
		return false
	}
	if !overrideExplicit {
		explicit = false
	}
	reflectResources := core.GetResourcesReflectively(resource)

	// Check to see if there are upstream nodes for the resource trying to be deleted
	// If upstream nodes exist, attempt to delete the resources upstream of the resource before deciding that the deletion process cannot continue
	if deletionCriteria.RequiresNoUpstream && !explicit && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.Id())
		if !e.ignoreCriteria(resource, upstreamNodes) {
			return false
		}
		for _, up := range upstreamNodes {
			e.deleteResource(up, false, false)
		}
		if len(e.KnowledgeBase.GetTrueUpstream(resource, graph)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.Id())
			return false
		}
	}
	if deletionCriteria.RequiresNoDownstream && !explicit && len(downstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
		if !e.ignoreCriteria(resource, downstreamNodes) {
			return false
		}
		for _, down := range downstreamNodes {
			e.deleteResource(down, false, false)
		}
		if len(e.KnowledgeBase.GetTrueDownstream(resource, graph)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
			return false
		}
	}
	if deletionCriteria.RequiresNoUpstreamOrDownstream && !explicit && len(downstreamNodes) > 0 && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
		if !e.ignoreCriteria(resource, upstreamNodes) && !e.ignoreCriteria(resource, downstreamNodes) {
			return false
		}
		for _, down := range downstreamNodes {
			e.deleteResource(down, false, false)
		}
		for _, up := range upstreamNodes {
			e.deleteResource(up, false, false)
		}
		if len(e.KnowledgeBase.GetTrueDownstream(resource, graph)) > 0 && len(e.KnowledgeBase.GetTrueUpstream(resource, graph)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream and downstream dependencies", resource.Id())
			return false
		}
	}
	err := graph.RemoveResourceAndEdges(resource)
	if err != nil {
		return false
	}

	for _, upstreamNode := range upstreamNodes {
		for _, downstreamNode := range downstreamNodes {

			var explicitUpstreams []core.Resource
			if upstreamNode.DeleteContext().RequiresExplicitDelete {
				explicitUpstreams = append(explicitUpstreams, upstreamNode)
			} else {
				explicitUpstreams = append(explicitUpstreams, e.getExplicitUpstreams(upstreamNode)...)
			}
			var explicitDownStreams []core.Resource
			if downstreamNode.DeleteContext().RequiresExplicitDelete {
				explicitDownStreams = append(explicitDownStreams, downstreamNode)
			} else {
				explicitDownStreams = append(explicitDownStreams, e.getExplicitDownStreams(downstreamNode)...)
			}

		UP:
			for _, u := range explicitUpstreams {
			DOWN:
				for _, d := range explicitDownStreams {

					for _, reflectResource := range reflectResources {
						if d == reflectResource {
							continue DOWN
						}
						if u == reflectResource {
							continue UP
						}
					}
					if len(e.KnowledgeBase.FindPathsInGraph(u, d, e.Context.EndState)) == 0 {
						log.Debugf("Adding dependency between %s and %s resources to reconnect path", u.Id(), d.Id())
						graph.AddDependency(u, d)
						e.Context.Constraints[constraints.EdgeConstraintScope] = append(e.Context.Constraints[constraints.EdgeConstraintScope],
							&constraints.EdgeConstraint{
								Operator: constraints.MustNotContainConstraintOperator,
								Target: constraints.Edge{
									Source: u.Id(),
									Target: d.Id(),
								},
								Node: resource.Id(),
							},
						)
					}
				}
			}
		}
	}

	for _, res := range upstreamNodes {
		explicit := false
		for _, reflectResource := range reflectResources {
			if res == reflectResource {
				explicit = true
				continue
			}
		}
		e.deleteResource(res, explicit, false)
	}
	for _, res := range downstreamNodes {
		explicit := false
		for _, reflectResource := range reflectResources {
			if res == reflectResource {
				explicit = true
				continue
			}
		}
		e.deleteResource(res, explicit, false)
	}
	return true
}

func (e *Engine) getExplicitUpstreams(res core.Resource) []core.Resource {
	var firstExplicitUpstreams []core.Resource
	upstreams := e.KnowledgeBase.GetTrueUpstream(res, e.Context.EndState)
	if len(upstreams) == 0 {
		return firstExplicitUpstreams
	}
	for _, up := range upstreams {
		if up.DeleteContext().RequiresExplicitDelete {
			firstExplicitUpstreams = append(firstExplicitUpstreams, up)
		}
	}
	if len(firstExplicitUpstreams) == 0 {
		for _, up := range upstreams {
			firstExplicitUpstreams = append(firstExplicitUpstreams, e.getExplicitUpstreams(up)...)
		}
	}
	return firstExplicitUpstreams
}

func (e *Engine) getExplicitDownStreams(res core.Resource) []core.Resource {
	var resources []core.Resource
	downstreams := e.KnowledgeBase.GetTrueDownstream(res, e.Context.EndState)
	if len(downstreams) == 0 {
		return resources
	}
	for _, d := range downstreams {
		if d.DeleteContext().RequiresExplicitDelete {
			resources = append(resources, d)
		}
	}
	if len(resources) == 0 {
		for _, up := range downstreams {
			resources = append(resources, e.getExplicitDownStreams(up)...)
		}
	}
	return resources
}

// handleOperationalResourceError tries to determine how to fix OperatioanlResourceErrors by adding dependencies to the resource graph where needed.
// If the error cannot be fixed, it will return an error.
func (e *Engine) handleOperationalResourceError(err *core.OperationalResourceError, dag *core.ResourceGraph) error {
	resources := e.ListResources()

	var neededResource core.Resource
	for _, res := range resources {
		if collectionutil.Contains(err.Needs, string(core.GetFunctionality(res))) {
			_, found := e.KnowledgeBase.GetEdge(err.Resource, res)
			if !found {
				continue
			}
			if neededResource != nil {
				return fmt.Errorf("multiple resources found that can satisfy the operational resource error, %s", err.Error())
			}
			neededResource = res
		}
	}
	if neededResource == nil {
		return fmt.Errorf("no resources found that can satisfy the operational resource error, %s", err.Error())
	}
	var availableResources []core.Resource
	for _, res := range dag.ListResources() {
		if res.Id().Type == neededResource.Id().Type {
			availableResources = append(availableResources, res)
		}
	}
	if len(availableResources) == 0 {
		reflect.ValueOf(neededResource).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%s", neededResource.Id().Type, err.Resource.Id().Name)))
		dag.AddDependency(err.Resource, neededResource)
	} else {
		resourceIds := []string{}
		for _, res := range availableResources {
			resourceIds = append(resourceIds, res.Id().Name)
		}
		sort.Strings(resourceIds)
		for _, res := range availableResources {
			if res.Id().Name == resourceIds[0] {
				dag.AddDependency(err.Resource, res)
				break
			}
		}
	}
	return nil
}
