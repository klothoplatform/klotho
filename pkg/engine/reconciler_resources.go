package engine

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"go.uber.org/zap"
)

// ignoreCriteriaR like ignoreCriteria but for resources and their graphs
func (e *Engine) ignoreCriteriaR(graph *construct.ResourceGraph, resource construct.Resource, dependentResources []construct.Resource) bool {
DEP:
	for _, dep := range dependentResources {
		for _, res := range graph.GetDownstreamResources(resource) {
			if dep == res {
				det, _ := e.KnowledgeBase.GetResourceEdge(resource, dep)
				if !det.DeletetionDependent {
					return false
				}
				continue DEP
			}
		}
		for _, res := range graph.GetUpstreamResources(resource) {
			if dep == res {
				det, _ := e.KnowledgeBase.GetResourceEdge(dep, resource)
				if !det.DeletetionDependent {
					return false
				}
				continue DEP
			}
		}
		return false
	}
	return true
}

// delete resource is used by the engine to remove resources from the resource graph in its context.
//
// if explicit is set it is meant to show that a user has explicitly requested for the resource to be deleted or that the resource requested is being deleted by its parent resource
// if overrideExplicit is set, it means that the explicit delete request still has to satisfy the resources delete criteria. If it is set to false, then the explicit deletion request is always performed
func (e *Engine) deleteResource(graph *construct.ResourceGraph, resource construct.Resource, explicit bool, overrideExplicit bool) bool {
	log := zap.S().With(zap.String("id", resource.Id().String()))
	log.Debug("Deleting resource")
	upstreamNodes := graph.GetUpstreamResources(resource)
	downstreamNodes := graph.GetDownstreamResources(resource)

	reflectResources := make(map[construct.ResourceId]construct.Resource)
	if !e.canDeleteResourceR(graph, resource, explicit, overrideExplicit, upstreamNodes, downstreamNodes) {
		return false
	}

	for _, reflectResource := range construct.GetResourcesReflectively(graph, resource) {
		reflectResources[reflectResource.Id()] = reflectResource
	}

	err := graph.RemoveResourceAndEdges(resource)
	if err != nil {
		return false
	}

	for _, upstreamNode := range upstreamNodes {
		if _, ok := reflectResources[upstreamNode.Id()]; ok {
			continue
		}

		var explicitUpstreams []construct.Resource
		if upstreamNode.DeleteContext().RequiresExplicitDelete {
			explicitUpstreams = append(explicitUpstreams, upstreamNode)
		} else {
			explicitUpstreams = append(explicitUpstreams, getExplicitDeleteUpstreamsR(graph, upstreamNode)...)
		}

		for _, downstreamNode := range downstreamNodes {
			if _, ok := reflectResources[downstreamNode.Id()]; ok {
				continue
			}

			var explicitDownStreams []construct.Resource
			if downstreamNode.DeleteContext().RequiresExplicitDelete {
				explicitDownStreams = append(explicitDownStreams, downstreamNode)
			} else {
				explicitDownStreams = append(explicitDownStreams, getExplicitDeleteDownstreamsR(graph, downstreamNode)...)
			}

			for _, u := range explicitUpstreams {
				for _, d := range explicitDownStreams {
					paths, err := graph.AllPaths(u.Id(), d.Id())
					if err != nil {
						zap.S().Debugf("Error getting paths between %s and %s", u.Id(), d.Id())
						continue
					}
					if len(paths) == 0 {
						log.Debugf("Adding dependency between %s and %s resources to reconnect path", u.Id(), d.Id())
						graph.AddDependencyById(u.Id(), d.Id(), nil)
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
		_, explicit := reflectResources[res.Id()]
		e.deleteResource(graph, res, explicit, false)
	}
	for _, res := range downstreamNodes {
		_, explicit := reflectResources[res.Id()]
		e.deleteResource(graph, res, explicit, false)
	}
	return true
}

func (e *Engine) canDeleteResourceR(graph *construct.ResourceGraph, resource construct.Resource, explicit bool, overrideExplicit bool, upstreamNodes []construct.Resource, downstreamNodes []construct.Resource) bool {
	log := zap.S().With(zap.String("id", resource.Id().String()))
	deletionCriteria := resource.DeleteContext()
	if deletionCriteria.RequiresExplicitDelete && !explicit {
		return false
	}
	if !overrideExplicit {
		explicit = false
	}
	// Check to see if there are upstream nodes for the resource trying to be deleted
	// If upstream nodes exist, attempt to delete the resources upstream of the resource before deciding that the deletion process cannot continue
	if deletionCriteria.RequiresNoUpstream && !explicit && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.Id())
		if !e.ignoreCriteriaR(graph, resource, upstreamNodes) {
			return false
		}
		for _, up := range upstreamNodes {
			e.deleteResource(graph, up, false, false)
		}
		if len(graph.GetUpstreamResources(resource)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.Id())
			return false
		}
	}
	if deletionCriteria.RequiresNoDownstream && !explicit && len(downstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
		if !e.ignoreCriteriaR(graph, resource, downstreamNodes) {
			return false
		}
		for _, down := range downstreamNodes {
			e.deleteResource(graph, down, false, false)
		}
		if len(graph.GetDownstreamResources(resource)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
			return false
		}
	}
	if deletionCriteria.RequiresNoUpstreamOrDownstream && !explicit && len(downstreamNodes) > 0 && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
		if !e.ignoreCriteriaR(graph, resource, upstreamNodes) && !e.ignoreCriteriaR(graph, resource, downstreamNodes) {
			return false
		}
		for _, down := range downstreamNodes {
			e.deleteResource(graph, down, false, false)
		}
		for _, up := range upstreamNodes {
			e.deleteResource(graph, up, false, false)
		}
		if len(graph.GetDownstreamResources(resource)) > 0 && len(graph.GetUpstreamResources(resource)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream and downstream dependencies", resource.Id())
			return false
		}
	}
	return true
}

func getExplicitDeleteUpstreamsR(graph *construct.ResourceGraph, res construct.Resource) []construct.Resource {
	var resources []construct.Resource
	upstreams := graph.GetUpstreamResources(res)
	if len(upstreams) == 0 {
		return resources
	}
	for _, up := range upstreams {
		if up.DeleteContext().RequiresExplicitDelete {
			resources = append(resources, up)
		}
	}
	if len(resources) == 0 {
		for _, up := range upstreams {
			resources = append(resources, getExplicitDeleteUpstreamsR(graph, up)...)
		}
	}
	return resources
}

func getExplicitDeleteDownstreamsR(graph *construct.ResourceGraph, res construct.Resource) []construct.Resource {
	var resources []construct.Resource
	downstreams := graph.GetDownstreamResources(res)
	if len(downstreams) == 0 {
		return resources
	}
	for _, d := range downstreams {
		if d.DeleteContext().RequiresExplicitDelete {
			resources = append(resources, d)
		}
	}
	if len(resources) == 0 {
		for _, down := range downstreams {
			resources = append(resources, getExplicitDeleteDownstreamsR(graph, down)...)
		}
	}
	return resources
}
