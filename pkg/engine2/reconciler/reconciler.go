package reconciler

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

func RemoveResource(c solution_context.SolutionContext, resource construct.ResourceId, explicit bool) error {

	upstreams, downstreams, err := construct.Neighbors(c.DataflowGraph(), resource)
	if err != nil {
		return err
	}

	template, err := c.KnowledgeBase().GetResourceTemplate(resource)
	if err != nil {
		return fmt.Errorf("unable to remove resource: error getting resource template for %s: %v", resource, err)
	}
	canDelete, err := canDeleteResource(c, resource, explicit, template, upstreams, downstreams)
	if err != nil {
		return err
	}
	if !canDelete {
		return nil
	}

	if template.GetFunctionality() == knowledgebase.Unknown {
		err := reconnectFunctionalResources(c, resource)
		if err != nil {
			return err
		}
	}

	op := c.OperationalView()

	var errs error
	// We must remove all edges before removing the vertex
	for res := range upstreams {
		errs = errors.Join(errs, op.RemoveEdge(res, resource))
	}
	for res := range downstreams {
		errs = errors.Join(errs, op.RemoveEdge(resource, res))
	}
	if errs != nil {
		return errs
	}

	err = construct.RemoveResource(op, resource)
	if err != nil {
		return err
	}

	// try to cleanup, if the resource is removable
	for res := range upstreams.Union(downstreams) {
		errs = errors.Join(errs, RemoveResource(c, res, false))
	}
	return nil
}

func reconnectFunctionalResources(ctx solution_context.SolutionContext, resource construct.ResourceId) error {
	log := zap.S().With(zap.String("id", resource.String()))
	functionalUpstreams, err := knowledgebase.UpstreamFunctional(ctx.DataflowGraph(), ctx.KnowledgeBase(), resource)
	ctxConstraints := ctx.Constraints()
	if err != nil {
		log.Errorf("Error getting functional upstreams for resource %s", resource)
		return err
	}
	functionalDownstreams, err := solution_context.DownstreamFunctional(ctx, resource)
	if err != nil {
		log.Errorf("Error getting functional downstreams for resource %s", resource)
		return err
	}
	for _, u := range functionalUpstreams {
		for _, d := range functionalDownstreams {
			paths, err := graph.AllPathsBetween(ctx.DataflowGraph(), u, d)
			if err != nil {
				zap.S().Debugf("Error getting paths between %s and %s", u, d)
				continue
			}
			var pathsWithoutRes [][]construct.ResourceId
		PATHS:
			for _, path := range paths {
				for _, res := range path {
					if res == resource {
						continue PATHS
					}
				}
				pathsWithoutRes = append(pathsWithoutRes, path)
			}
			if len(pathsWithoutRes) == 0 {
				log.Debugf("Adding dependency between %s and %s resources to reconnect path", u, d)
				ctxConstraints.Edges = append(ctxConstraints.Edges, constraints.EdgeConstraint{
					Operator: constraints.MustNotContainConstraintOperator,
					Target: constraints.Edge{
						Source: u,
						Target: d,
					},
					Node: resource,
				})
				err := ctx.OperationalView().AddEdge(u, d)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func canDeleteResource(
	ctx solution_context.SolutionContext,
	resource construct.ResourceId,
	explicit bool,
	template *knowledgebase.ResourceTemplate,
	upstreamNodes set.Set[construct.ResourceId],
	downstreamNodes set.Set[construct.ResourceId],
) (bool, error) {
	if template.GetFunctionality() != knowledgebase.Unknown && !explicit {
		return false, nil
	}

	log := zap.S().With(zap.String("id", resource.String()))
	deletionCriteria := template.DeleteContext

	ignoreUpstream, err := ignoreCriteria(ctx, resource, upstreamNodes)
	if err != nil {
		return false, err
	}
	ignoreDownstream, err := ignoreCriteria(ctx, resource, downstreamNodes)
	if err != nil {
		return false, err
	}

	// Check to see if there are upstream nodes for the resource trying to be deleted
	// If upstream nodes exist, attempt to delete the resources upstream of the resource before deciding that the deletion process cannot continue
	if deletionCriteria.RequiresNoUpstream && !explicit && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource)
		if !ignoreUpstream {
			return false, nil
		}
		for up := range upstreamNodes {
			err := RemoveResource(ctx, up, false)
			if err != nil {
				return false, err
			}
		}
		// Now that we have attempted to delete the upstream resources, check to see if there are any upstream resources left for the deletion criteria
		upstream, err := construct.DirectUpstreamDependencies(ctx.DataflowGraph(), resource)
		if err != nil {
			return false, err
		}
		if len(upstream) > 0 {
			return false, fmt.Errorf("cannot delete resource %s as it still has %d upstream dependencies", resource, len(upstream))
		}
	}
	if deletionCriteria.RequiresNoDownstream && !explicit && len(downstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource)
		if !ignoreDownstream {
			return false, nil
		}
		for down := range downstreamNodes {
			err := RemoveResource(ctx, down, false)
			if err != nil {
				return false, err
			}
		}
		// Now that we have attempted to delete the downstream resources, check to see if there are any downstream resources left for the deletion criteria
		downstream, err := construct.DirectDownstreamDependencies(ctx.DataflowGraph(), resource)
		if err != nil {
			return false, err
		}
		if len(downstream) > 0 {
			return false, fmt.Errorf("cannot delete resource %s as it still has %d downstream dependencies", resource, len(downstream))
		}
	}
	if deletionCriteria.RequiresNoUpstreamOrDownstream && !explicit && len(downstreamNodes) > 0 && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource)
		if !ignoreUpstream && !ignoreDownstream {
			return false, nil
		}
		for down := range downstreamNodes {
			err := RemoveResource(ctx, down, false)
			if err != nil {
				return false, err
			}
		}
		for up := range upstreamNodes {
			err := RemoveResource(ctx, up, false)
			if err != nil {
				return false, err
			}
		}
		// Now that we have attempted to delete the downstream resources, check to see if there are any downstream resources left for the deletion criteria
		downstream, err := construct.DirectDownstreamDependencies(ctx.DataflowGraph(), resource)
		if err != nil {
			return false, err
		}
		// Now that we have attempted to delete the upstream resources, check to see if there are any upstream resources left for the deletion criteria
		upstream, err := construct.DirectUpstreamDependencies(ctx.DataflowGraph(), resource)
		if err != nil {
			return false, err
		}
		if len(downstream) > 0 && len(upstream) > 0 {
			return false, fmt.Errorf(
				"cannot delete resource %s as it still has %d upstream and %d downstream dependencies",
				resource,
				len(upstream),
				len(downstream),
			)
		}
	}
	return true, nil
}

// ignoreCriteria determines if we can delete a resource because the knowledge base in use by the engine, shows that the initial resource is dependent on the sub resource for deletion.
// If the sub resource is deletion dependent on any of the dependent resources passed in then we will determine weather we can delete the dependent resource first.
func ignoreCriteria(ctx solution_context.SolutionContext, resource construct.ResourceId, dependentResources set.Set[construct.ResourceId]) (bool, error) {
	upstreams, downstreams, err := construct.Neighbors(ctx.DataflowGraph(), resource)
	if err != nil {
		return false, err
	}

	upstreams = upstreams.Intersection(dependentResources)
	downstreams = downstreams.Intersection(dependentResources)

	for up := range upstreams {
		t := ctx.KnowledgeBase().GetEdgeTemplate(up, resource)
		if !t.DeletetionDependent {
			return false, nil
		}
	}
	for down := range downstreams {
		t := ctx.KnowledgeBase().GetEdgeTemplate(resource, down)
		if !t.DeletetionDependent {
			return false, nil
		}
	}
	return true, nil
}
