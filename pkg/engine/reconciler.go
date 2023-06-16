package engine

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"go.uber.org/zap"
)

func (e *Engine) deleteResource(resource core.Resource, explicit bool) bool {
	log := zap.S().With(zap.String("id", resource.Id().String()))
	log.Debug("Deleting resource")
	graph := e.Context.EndState
	deletionCriteria := resource.DeleteCriteria()
	upstreamNodes := e.KnowledgeBase.GetTrueUpstream(resource, graph)
	downstreamNodes := e.KnowledgeBase.GetTrueDownstream(resource, graph)
	if deletionCriteria.RequiresExplicitDelete && !explicit {
		log.Debug("Cannot delete resource as it was not explicitly requested")
		return false
	}
	if deletionCriteria.RequiresNoUpstream && !explicit && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.Id())
		return false
	}
	if deletionCriteria.RequiresNoDownstream && !explicit && len(downstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.Id())
		return false
	}
	if deletionCriteria.RequiresNoUpstreamOrDownstream && !explicit && len(downstreamNodes) > 0 && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has upstream and downstream dependencies", resource.Id())
		return false
	}
	graph.RemoveResourceAndEdges(resource)

	for _, upstreamNode := range upstreamNodes {
		for _, downstreamNode := range downstreamNodes {

			var explicitUpstreams []core.Resource
			if upstreamNode.DeleteCriteria().RequiresExplicitDelete {
				explicitUpstreams = append(explicitUpstreams, upstreamNode)
			} else {
				explicitUpstreams = append(explicitUpstreams, e.getExplicitUpstreams(upstreamNode)...)
			}
			var explicitDownStreams []core.Resource
			if downstreamNode.DeleteCriteria().RequiresExplicitDelete {
				explicitDownStreams = append(explicitDownStreams, downstreamNode)
			} else {
				explicitDownStreams = append(explicitDownStreams, e.getExplicitDownStreams(downstreamNode)...)
			}
			for _, u := range explicitUpstreams {
				for _, d := range explicitDownStreams {
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

	for _, upstreamNode := range upstreamNodes {
		res, ok := upstreamNode.(core.Resource)
		if !ok {
			continue
		}
		e.deleteResource(res, false)
	}
	for _, downstreamNode := range downstreamNodes {
		res, ok := downstreamNode.(core.Resource)
		if !ok {
			continue
		}
		e.deleteResource(res, false)
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
		if up.DeleteCriteria().RequiresExplicitDelete {
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
		if d.DeleteCriteria().RequiresExplicitDelete {
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
