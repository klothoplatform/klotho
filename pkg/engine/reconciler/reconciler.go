package reconciler

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type (
	deleteRequest struct {
		resource construct.ResourceId
		explicit bool
	}
)

func RemoveResource(c solution_context.SolutionContext, resource construct.ResourceId, explicit bool) error {
	zap.S().Debugf("reconciling removal of resource %s ", resource)

	queue := []deleteRequest{{
		resource: resource,
		explicit: explicit,
	}}

	var errs error

	for len(queue) > 0 {
		request := queue[0]
		queue = queue[1:]
		resource := request.resource
		explicit := request.explicit

		upstreams, downstreams, err := construct.Neighbors(c.DataflowGraph(), resource)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		template, err := c.KnowledgeBase().GetResourceTemplate(resource)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("unable to remove resource: error getting resource template for %s: %v", resource, err))
			continue
		}
		canDelete, err := canDeleteResource(c, resource, explicit, template, upstreams, downstreams)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if !canDelete {
			continue
		}
		// find all namespaced resources before removing edges and the initial resource, otherwise certain resources may
		// be moved out of their original namespace
		namespacedResources, err := findAllResourcesInNamespace(c, resource)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for res := range namespacedResources {
			// Since we may be explicitly deleting the namespace resource,
			// we will forward the same explicit flag to the namespace resource
			queue = appendToQueue(deleteRequest{resource: res, explicit: explicit}, queue)
			// find deployment dependencies to ensure the resource wont get recreated
			queue, err = addAllDeploymentDependencies(c, res, explicit, queue)
			if err != nil {
				errs = errors.Join(errs, err)
			}
		}

		op := c.OperationalView()

		// We must remove all edges before removing the vertex
		for res := range upstreams {
			errs = errors.Join(errs, op.RemoveEdge(res, resource))
		}
		for res := range downstreams {
			errs = errors.Join(errs, op.RemoveEdge(resource, res))
		}
		errs = errors.Join(errs, deleteRemainingDeploymentDependencies(c, resource))

		err = construct.RemoveResource(op, resource)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		// try to cleanup, if the resource is removable
		for res := range upstreams.Union(downstreams) {
			queue = appendToQueue(deleteRequest{resource: res, explicit: false}, queue)
		}
	}
	return errs
}

func appendToQueue(request deleteRequest, queue []deleteRequest) []deleteRequest {
	for i, req := range queue {
		if req.resource == request.resource && req.explicit == request.explicit {
			return queue
			// only update if it wasnt explicit previously and now it is
		} else if req.resource == request.resource && !req.explicit {
			queue[i] = request
			return queue
		} else if req.resource == request.resource && req.explicit {
			return queue
		}
	}
	return append(queue, request)
}

// deleteRemainingDeploymentDependencies removes all deployment dependencies from the graph for the resource specified
func deleteRemainingDeploymentDependencies(
	ctx solution_context.SolutionContext,
	resource construct.ResourceId,
) error {
	// begin by removing the dependency on resource since we know we are deleting the resource passed in at this point
	var errs error
	upstreamDeploymentDeps, err := construct.DirectUpstreamDependencies(ctx.DeploymentGraph(), resource)
	if err != nil {
		errs = errors.Join(errs, err)
	} else {
		for _, res := range upstreamDeploymentDeps {
			err = ctx.DeploymentGraph().RemoveEdge(res, resource)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		}
	}

	downstreamDeploymentDeps, err := construct.DirectDownstreamDependencies(ctx.DeploymentGraph(), resource)
	if err != nil {
		errs = errors.Join(errs, err)
	} else {
		for _, res := range downstreamDeploymentDeps {
			err = ctx.DeploymentGraph().RemoveEdge(resource, res)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		}
	}
	return errs
}

// addAllDeploymentDependencies adds all deployment dependencies to the queue while removing their dependency on the resource passed in
func addAllDeploymentDependencies(
	ctx solution_context.SolutionContext,
	resource construct.ResourceId,
	explicit bool,
	queue []deleteRequest,
) ([]deleteRequest, error) {

	deploymentDeps, err := knowledgebase.Upstream(
		ctx.DeploymentGraph(),
		ctx.KnowledgeBase(),
		resource,
		knowledgebase.ResourceDirectLayer,
	)
	if err != nil {
		return nil, err
	}

	for _, dep := range deploymentDeps {
		res, err := ctx.RawView().Vertex(dep)
		if err != nil {
			return nil, err
		}
		rt, err := ctx.KnowledgeBase().GetResourceTemplate(dep)
		if err != nil {
			return nil, err
		}
		if collectionutil.Contains(queue, deleteRequest{resource: dep, explicit: explicit}) {
			continue
		}

		shouldDelete := false

		// check if the dep exists as a property on the resource
		err = rt.LoopProperties(res, func(p knowledgebase.Property) error {
			propVal, err := res.GetProperty(p.Details().Path)
			if err != nil {
				return err
			}
			found := false
			switch val := propVal.(type) {
			case construct.ResourceId:
				if val == resource {
					found = true
				}
			case construct.PropertyRef:
				if val.Resource == resource {
					found = true
				}
			default:
				if reflect.ValueOf(val).Kind() == reflect.Slice || reflect.ValueOf(val).Kind() == reflect.Array {
					for i := 0; i < reflect.ValueOf(val).Len(); i++ {
						idVal := reflect.ValueOf(val).Index(i).Interface()
						if id, ok := idVal.(construct.ResourceId); ok && id == resource {
							found = true
						} else if pref, ok := idVal.(construct.PropertyRef); ok && pref.Resource == resource {
							found = true
						}
					}
				}
			}
			if found {
				if p.Details().OperationalRule != nil || p.Details().Required {
					shouldDelete = true
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if shouldDelete {
			queue = appendToQueue(deleteRequest{resource: dep, explicit: explicit}, queue)
			queue, err = addAllDeploymentDependencies(ctx, dep, explicit, queue)
			if err != nil {
				return nil, err
			}
		}
	}
	return queue, nil
}

func canDeleteResource(
	ctx solution_context.SolutionContext,
	resource construct.ResourceId,
	explicit bool,
	template *knowledgebase.ResourceTemplate,
	upstreamNodes set.Set[construct.ResourceId],
	downstreamNodes set.Set[construct.ResourceId],
) (bool, error) {

	res, err := ctx.RawView().Vertex(resource)
	if err != nil {
		return false, err
	}
	// dont allow deletion of imported or functional resources unless it is explicitly stated
	if (template.GetFunctionality() != knowledgebase.Unknown || res.Imported) && !explicit {
		return false, nil
	}

	log := zap.S().With(zap.String("id", resource.String()))
	deletionCriteria := template.DeleteContext

	ignoreUpstream := ignoreCriteria(ctx, resource, upstreamNodes, knowledgebase.DirectionUpstream)
	ignoreDownstream := ignoreCriteria(ctx, resource, downstreamNodes, knowledgebase.DirectionDownstream)

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

// ignoreCriteria determines if we can delete a resource because the knowledge base in use by the engine,
// shows that the initial resource is dependent on the sub resource for deletion.
// If the sub resource is deletion dependent on any of the dependent resources passed in then we will determine weather
// we can delete the dependent resource first.
func ignoreCriteria(
	ctx solution_context.SolutionContext,
	resource construct.ResourceId,
	nodes set.Set[construct.ResourceId],
	direction knowledgebase.Direction,
) bool {
	if direction == knowledgebase.DirectionDownstream {
		for down := range nodes {
			t := ctx.KnowledgeBase().GetEdgeTemplate(resource, down)
			if t == nil || !t.DeletionDependent {
				return false
			}
		}
	} else {
		for up := range nodes {
			t := ctx.KnowledgeBase().GetEdgeTemplate(up, resource)
			if t == nil || !t.DeletionDependent {
				return false
			}
		}
	}
	return true
}

func findAllResourcesInNamespace(ctx solution_context.SolutionContext, namespace construct.ResourceId) (set.Set[construct.ResourceId], error) {
	namespacedResources := make(set.Set[construct.ResourceId])
	err := construct.WalkGraph(ctx.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if id.Namespace == "" || id.Namespace != namespace.Name {
			return nerr
		}
		rt, err := ctx.KnowledgeBase().GetResourceTemplate(id)
		if err != nil {
			return errors.Join(nerr, err)
		}
		if rt == nil {
			return errors.Join(nerr, fmt.Errorf("unable to find resource template for %s", id))
		}
		err = rt.LoopProperties(resource, func(p knowledgebase.Property) error {
			if !p.Details().Namespace {
				return nil
			}
			propVal, err := resource.GetProperty(p.Details().Path)
			if err != nil {
				return err
			}
			switch val := propVal.(type) {
			case construct.ResourceId:
				if val.Matches(namespace) {
					namespacedResources.Add(id)
				}
			case construct.PropertyRef:
				if val.Resource.Matches(namespace) {
					namespacedResources.Add(id)
				}
			}
			return nil
		})
		return errors.Join(nerr, err)
	})
	if err != nil {
		return nil, err
	}
	return namespacedResources, nil
}
