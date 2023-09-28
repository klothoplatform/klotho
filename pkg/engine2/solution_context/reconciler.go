package solution_context

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

func (ctx SolutionContext) reconnectFunctionalResources(resource *construct.Resource) error {
	log := zap.S().With(zap.String("id", resource.ID.String()))
	functionalUpstreams, err := ctx.UpstreamFunctional(resource)
	if err != nil {
		log.Errorf("Error getting functional upstreams for resource %s", resource.ID)
		return err
	}
	functionalDownstreams, err := ctx.DownstreamFunctional(resource)
	if err != nil {
		log.Errorf("Error getting functional downstreams for resource %s", resource.ID)
		return err
	}
	for _, u := range functionalUpstreams {
		for _, d := range functionalDownstreams {

			paths, err := ctx.AllPaths(u.ID, d.ID)
			if err != nil {
				zap.S().Debugf("Error getting paths between %s and %s", u.ID, d.ID)
				continue
			}
			if len(paths) == 0 {
				log.Debugf("Adding dependency between %s and %s resources to reconnect path", u.ID, d.ID)
				ctx.EdgeConstraints = append(ctx.EdgeConstraints, constraints.EdgeConstraint{
					Operator: constraints.MustNotContainConstraintOperator,
					Target: constraints.Edge{
						Source: u.ID,
						Target: d.ID,
					},
					Node: resource.ID,
				})
				err := ctx.AddDependency(u, d)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ctx SolutionContext) canDeleteResource(resource *construct.Resource, explicit bool, upstreamNodes []*construct.Resource, downstreamNodes []*construct.Resource) bool {
	log := zap.S().With(zap.String("id", resource.ID.String()))
	template, err := ctx.kb.GetResourceTemplate(resource.ID)
	if err != nil {
		log.Errorf("Unable to get resource template for resource %s", resource.ID)
		return false
	}
	deletionCriteria := template.DeleteContext
	if template.GetFunctionality() != knowledgebase.Unknown && !explicit {
		return false
	}

	// Check to see if there are upstream nodes for the resource trying to be deleted
	// If upstream nodes exist, attempt to delete the resources upstream of the resource before deciding that the deletion process cannot continue
	if deletionCriteria.RequiresNoUpstream && !explicit && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.ID)
		if !ctx.ignoreCriteria(resource, upstreamNodes) {
			return false
		}
		for _, up := range upstreamNodes {
			err := ctx.RemoveResource(up, false)
			if err != nil {
				log.Errorf("Unable to delete upstream resource %s for resource %s", up.ID, resource.ID)
				return false
			}
		}
		// Now that we have attempted to delete the upstream resources, check to see if there are any upstream resources left for the deletion criteria
		upstream, err := ctx.DirectUpstreamResources(resource.ID)
		if err != nil {
			log.Errorf("Unable to get upstream resources for resource %s", resource.ID)
			return false
		}
		if len(upstream) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream dependencies", resource.ID)
			return false
		}
	}
	if deletionCriteria.RequiresNoDownstream && !explicit && len(downstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.ID)
		if !ctx.ignoreCriteria(resource, downstreamNodes) {
			return false
		}
		for _, down := range downstreamNodes {
			err = ctx.RemoveResource(down, false)
			if err != nil {
				log.Errorf("Unable to delete downstream resource %s for resource %s", down.ID, resource.ID)
				return false
			}
		}
		// Now that we have attempted to delete the downstream resources, check to see if there are any downstream resources left for the deletion criteria
		downstream, err := ctx.DirectDownstreamResources(resource.ID)
		if err != nil {
			log.Errorf("Unable to get downstream resources for resource %s", resource.ID)
			return false
		}
		if len(downstream) > 0 {
			log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.ID)
			return false
		}
	}
	if deletionCriteria.RequiresNoUpstreamOrDownstream && !explicit && len(downstreamNodes) > 0 && len(upstreamNodes) > 0 {
		log.Debugf("Cannot delete resource %s as it still has downstream dependencies", resource.ID)
		if !ctx.ignoreCriteria(resource, upstreamNodes) && !ctx.ignoreCriteria(resource, downstreamNodes) {
			return false
		}
		for _, down := range downstreamNodes {
			err = ctx.RemoveResource(down, false)
			if err != nil {
				log.Errorf("Unable to delete downstream resource %s for resource %s", down.ID, resource.ID)
				return false
			}
		}
		for _, up := range upstreamNodes {
			err = ctx.RemoveResource(up, false)
			if err != nil {
				log.Errorf("Unable to delete upstream resource %s for resource %s", up.ID, resource.ID)
				return false
			}
		}
		// Now that we have attempted to delete the downstream resources, check to see if there are any downstream resources left for the deletion criteria
		downstream, err := ctx.DirectDownstreamResources(resource.ID)
		if err != nil {
			log.Errorf("Unable to get downstream resources for resource %s", resource.ID)
			return false
		}
		// Now that we have attempted to delete the upstream resources, check to see if there are any upstream resources left for the deletion criteria
		upstream, err := ctx.DirectUpstreamResources(resource.ID)
		if err != nil {
			log.Errorf("Unable to get upstream resources for resource %s", resource.ID)
			return false
		}
		if len(downstream) > 0 && len(upstream) > 0 {
			log.Debugf("Cannot delete resource %s as it still has upstream and downstream dependencies", resource.ID)
			return false
		}
	}
	return true
}

// ignoreCriteria determines if we can delete a resource because the knowledge base in use by the engine, shows that the initial resource is dependent on the sub resource for deletion.
// If the sub resource is deletion dependent on any of the dependent resources passed in then we will determine weather we can delete the dependent resource first.
func (ctx SolutionContext) ignoreCriteria(resource *construct.Resource, dependentResources []*construct.Resource) bool {
DEP:
	for _, dep := range dependentResources {
		downstreams, err := ctx.DirectDownstreamResources(resource.ID)
		if err != nil {
			zap.S().Errorf("Unable to get downstream resources for resource %s", resource.ID)
			return false
		}
		for _, res := range downstreams {
			if dep == res {
				det := ctx.kb.GetEdgeTemplate(resource.ID, dep.ID)
				if !det.DeletetionDependent {
					return false
				}
				continue DEP
			}
		}
		upstreams, err := ctx.DirectUpstreamResources(resource.ID)
		if err != nil {
			zap.S().Errorf("Unable to get upstream resources for resource %s", resource.ID)
			return false
		}
		for _, res := range upstreams {
			if dep == res {
				det := ctx.kb.GetEdgeTemplate(dep.ID, resource.ID)
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
