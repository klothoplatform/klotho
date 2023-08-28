package engine

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"go.uber.org/zap"
)

// ignoreCriteria determines if we can delete a resource because the knowledge base in use by the engine, shows that the initial resource is dependent on the sub resource for deletion.
// If the sub resource is deletion dependent on any of the dependent resources passed in then we will determine weather we can delete the dependent resource first.
func (e *Engine) ignoreCriteria(resource construct.Resource, dependentResources []construct.BaseConstruct) bool {
DEP:
	for _, dep := range dependentResources {
		if _, ok := dep.(construct.Construct); ok {
			continue
		} else if dep, ok := dep.(construct.Resource); ok {
			found := false
			for _, res := range e.Context.WorkingState.GetDownstreamConstructs(resource) {
				if _, ok := res.(construct.Construct); ok {
					continue
				}
				if dep == res {
					found = true
					det, _ := e.KnowledgeBase.GetResourceEdge(resource, dep)
					if !det.DeletetionDependent {
						return false
					}
					continue DEP
				}
			}
			for _, res := range e.Context.WorkingState.GetUpstreamConstructs(resource) {
				if _, ok := res.(construct.Construct); ok {
					continue
				}
				if dep == res {
					found = true
					det, _ := e.KnowledgeBase.GetResourceEdge(dep, resource)
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
	}
	return true
}

// delete resource is used by the engine to remove resources from the resource graph in its context.
//
// if explicit is set it is meant to show that a user has explicitly requested for the resource to be deleted or that the resource requested is being deleted by its parent resource
// if overrideExplicit is set, it means that the explicit delete request still has to satisfy the resources delete criteria. If it is set to false, then the explicit deletion request is always performed
func (e *Engine) deleteConstruct(c construct.BaseConstruct, explicit bool, overrideExplicit bool) bool {
	log := zap.S().With(zap.String("id", c.Id().String()))
	log.Debug("Deleting resource")
	graph := e.Context.WorkingState
	upstreamNodes := e.Context.WorkingState.GetUpstreamConstructs(c)
	downstreamNodes := e.Context.WorkingState.GetDownstreamConstructs(c)

	var reflectResources []construct.Resource
	if resource, ok := c.(construct.Resource); ok {
		reflectResources = construct.GetResourcesReflectively(graph, resource)
		if !e.canDeleteResource(resource, explicit, overrideExplicit, upstreamNodes, downstreamNodes) {
			return false
		}
	} else if _, ok := c.(construct.Construct); ok {
		if !explicit {
			return false
		}
	}

	err := graph.RemoveConstructAndEdges(c)
	if err != nil {
		return false
	}

	for _, upstreamNode := range upstreamNodes {
		for _, downstreamNode := range downstreamNodes {

			var explicitUpstreams []construct.BaseConstruct
			if c, ok := upstreamNode.(construct.Construct); ok {
				explicitUpstreams = append(explicitUpstreams, c)
			} else if resource, ok := upstreamNode.(construct.Resource); ok {
				if resource.DeleteContext().RequiresExplicitDelete {
					explicitUpstreams = append(explicitUpstreams, resource)
				} else {
					explicitUpstreams = append(explicitUpstreams, e.getExplicitUpstreams(resource)...)
				}
			}
			var explicitDownStreams []construct.BaseConstruct
			if c, ok := downstreamNode.(construct.Construct); ok {
				explicitUpstreams = append(explicitUpstreams, c)
			} else if resource, ok := downstreamNode.(construct.Resource); ok {
				if resource.DeleteContext().RequiresExplicitDelete {
					explicitDownStreams = append(explicitDownStreams, downstreamNode)
				} else {
					explicitDownStreams = append(explicitDownStreams, e.getExplicitDownStreams(downstreamNode)...)
				}
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
					paths, err := e.Context.WorkingState.AllPaths(u.Id(), d.Id())
					if err != nil {
						zap.S().Debugf("Error getting paths between %s and %s", u.Id(), d.Id())
						continue
					}
					if len(paths) == 0 {
						log.Debugf("Adding dependency between %s and %s resources to reconnect path", u.Id(), d.Id())
						graph.AddDependency(u.Id(), d.Id())
						e.Context.Constraints[constraints.EdgeConstraintScope] = append(e.Context.Constraints[constraints.EdgeConstraintScope],
							&constraints.EdgeConstraint{
								Operator: constraints.MustNotContainConstraintOperator,
								Target: constraints.Edge{
									Source: u.Id(),
									Target: d.Id(),
								},
								Node: c.Id(),
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
		e.deleteConstruct(res, explicit, false)
	}
	for _, res := range downstreamNodes {
		explicit := false
		for _, reflectResource := range reflectResources {
			if res == reflectResource {
				explicit = true
				continue
			}
		}
		e.deleteConstruct(res, explicit, false)
	}
	return true
}

func (e *Engine) canDeleteResource(resource construct.Resource, explicit bool, overrideExplicit bool, upstreamNodes []construct.BaseConstruct, downstreamNodes []construct.BaseConstruct) bool {
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
		if !e.ignoreCriteria(resource, upstreamNodes) {
			return false
		}
		for _, up := range upstreamNodes {
			e.deleteConstruct(up, false, false)
		}
		if len(e.Context.WorkingState.GetUpstreamConstructs(resource)) > 0 {
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
			e.deleteConstruct(down, false, false)
		}
		if len(e.Context.WorkingState.GetDownstreamConstructs(resource)) > 0 {
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
			e.deleteConstruct(down, false, false)
		}
		for _, up := range upstreamNodes {
			e.deleteConstruct(up, false, false)
		}
		if len(e.Context.WorkingState.GetDownstreamConstructs(resource)) > 0 && len(e.Context.WorkingState.GetUpstreamConstructs(resource)) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream and downstream dependencies", resource.Id())
			return false
		}
	}
	return true
}

func (e *Engine) getExplicitUpstreams(res construct.BaseConstruct) []construct.BaseConstruct {
	var resources []construct.BaseConstruct
	upstreams := e.Context.WorkingState.GetUpstreamConstructs(res)
	if len(upstreams) == 0 {
		return resources
	}
	for _, up := range upstreams {
		if c, ok := up.(construct.Construct); ok {
			resources = append(resources, c)
		} else if resource, ok := up.(construct.Resource); ok {
			if resource.DeleteContext().RequiresExplicitDelete {
				resources = append(resources, up)
			}
		}
	}
	if len(resources) == 0 {
		for _, up := range upstreams {
			resources = append(resources, e.getExplicitUpstreams(up)...)
		}
	}
	return resources
}

func (e *Engine) getExplicitDownStreams(res construct.BaseConstruct) []construct.BaseConstruct {
	var resources []construct.BaseConstruct
	downstreams := e.Context.WorkingState.GetDownstreamConstructs(res)
	if len(downstreams) == 0 {
		return resources
	}
	for _, d := range downstreams {
		if c, ok := d.(construct.Construct); ok {
			resources = append(resources, c)
		} else if resource, ok := d.(construct.Resource); ok {
			if resource.DeleteContext().RequiresExplicitDelete {
				resources = append(resources, resource)
			}
		}
	}
	if len(resources) == 0 {
		for _, down := range downstreams {
			resources = append(resources, e.getExplicitDownStreams(down)...)
		}
	}
	return resources
}
