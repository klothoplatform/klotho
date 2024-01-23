package reconciler

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

// RemovePath removes all paths between the source and target node.
//
// It will determine when edges within those paths are used for contexts outside of the source and target paths and not remove them.
func RemovePath(
	source, target construct.ResourceId,
	ctx solution_context.SolutionContext,
) error {
	zap.S().Infof("Removing path %s -> %s", source, target)
	paths, err := graph.AllPathsBetween(ctx.DataflowGraph(), source, target)
	switch {
	case errors.Is(err, graph.ErrTargetNotReachable):
		return nil
	case err != nil:
		return err
	}

	nodes := nodesInPaths(paths)
	used, err := nodesUsedOutsideOfContext(nodes, ctx)
	if err != nil {
		return err
	}
	edges, err := findEdgesUsedInOtherPathSelection(source, target, nodes, ctx)
	if err != nil {
		return err
	}

	var errs error
	for _, path := range paths {
		errs = errors.Join(errs, removeSinglePath(source, target, path, used, edges, ctx))
	}

	// Next we will try to delete any node in those paths in case they no longer are required for the architecture
	// We will pass the explicit field as false so that explicitly added resources do not get deleted
	for _, path := range paths {
		for _, resource := range path {
			// by this point its possible the resource no longer exists due to being deleted by the removeSinglePath call
			// since this is ensuring we dont orphan resources we can ignore the error if we do not find the resource
			err := RemoveResource(ctx, resource, false)
			if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

// removeSinglePath removes all edges in a single path between the source and target node, if they are allowed to be removed.
//
// in order for an edge to be removed:
// The source of the edge must not have other upstream dependencies (signaling there are other paths its connecting)
// The edge must not be used in path solving another connection from the source or to the target
func removeSinglePath(
	source, target construct.ResourceId,
	path []construct.ResourceId,
	used set.Set[construct.ResourceId],
	edges set.Set[construct.SimpleEdge],
	ctx solution_context.SolutionContext,
) error {
	var errs error
	// first we will remove all dependencies that make up the paths from the constraints source to target
	for i, res := range path {
		if i == 0 {
			continue
		}
		// check if the previous resource is used outside of its context.
		// Since we are deleting the edge downstream we have to make sure the source is not used,
		// resulting in this edge being a part of another path
		if used.Contains(path[i-1]) && !target.Matches(path[i-1]) && !source.Matches(path[i-1]) {
			continue
		}
		if edges.Contains(construct.SimpleEdge{Source: path[i-1], Target: res}) {
			continue
		}
		errs = errors.Join(errs, ctx.OperationalView().RemoveEdge(path[i-1], res))
		if i > 1 {
			errs = errors.Join(errs, RemoveResource(ctx, path[i-1], false))
		}
	}
	return errs
}

func nodesInPaths(
	paths [][]construct.ResourceId,
) set.Set[construct.ResourceId] {
	nodes := make(set.Set[construct.ResourceId])
	for _, path := range paths {
		for _, res := range path {
			nodes.Add(res)
		}
	}
	return nodes
}

// nodesUsedOutsideOfContext returns all nodes that are used outside of the context.
// Being used outside of the context entails that there are upstream connections in the dataflow graph
// outside of the set of nodes used in the all paths between the source and target node.
//
// We only care about upstream because any extra connections downstream will stay intact and wont result in other
// paths being affected
func nodesUsedOutsideOfContext(
	nodes set.Set[construct.ResourceId],
	ctx solution_context.SolutionContext,
) (set.Set[construct.ResourceId], error) {
	var errs error
	used := make(set.Set[construct.ResourceId])
	pred, err := ctx.RawView().PredecessorMap()
	if err != nil {
		return nil, err
	}
	for node := range nodes {
		upstreams := pred[node]
		for upstream := range upstreams {
			if !nodes.Contains(upstream) {
				used.Add(node)
			}
		}
	}
	return used, errs
}

// findEdgesUsedInOtherPathSelection returns all edges that are used in other path selections to the target or from the source.
func findEdgesUsedInOtherPathSelection(
	source, target construct.ResourceId,
	nodes set.Set[construct.ResourceId],
	ctx solution_context.SolutionContext,
) (set.Set[construct.SimpleEdge], error) {
	edges := make(set.Set[construct.SimpleEdge])
	var errs error
	upstreams, err := construct.AllUpstreamDependencies(ctx.DataflowGraph(), target)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	for _, upstream := range upstreams {
		if nodes.Contains(upstream) {
			continue
		}
		upstreamRT, err := ctx.KnowledgeBase().GetResourceTemplate(upstream)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		} else if upstreamRT == nil {
			errs = errors.Join(errs, fmt.Errorf("resource template %s not found", upstream))
			continue
		}
		if len(upstreamRT.PathSatisfaction.AsSource) == 0 {
			continue
		}
		paths, err := path_selection.GetPaths(ctx, upstream, target,
			func(source, target construct.ResourceId, path construct.Path) bool { return true }, false)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, path := range paths {
			// if the source is in the path then the path is just a superset of the path we are trying to delete
			if path.Contains(source) {
				continue
			}
			for i, res := range path {
				if i == 0 {
					continue
				}
				edges.Add(construct.SimpleEdge{Source: path[i-1], Target: res})
			}
		}
	}
	downstreams, err := construct.AllDownstreamDependencies(ctx.DataflowGraph(), source)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	for _, downstream := range downstreams {
		if nodes.Contains(downstream) {
			continue
		}
		downstreamRT, err := ctx.KnowledgeBase().GetResourceTemplate(downstream)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		} else if downstreamRT == nil {
			errs = errors.Join(errs, fmt.Errorf("resource template %s not found", downstream))
			continue
		}
		if len(downstreamRT.PathSatisfaction.AsTarget) == 0 {
			continue
		}

		paths, err := path_selection.GetPaths(ctx, source, downstream,
			func(source, target construct.ResourceId, path construct.Path) bool { return true }, false)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		for _, path := range paths {
			// if the target is in the path then the path is just a superset of the path we are trying to delete
			if path.Contains(target) {
				continue
			}
			for i, res := range path {
				if i == 0 {
					continue
				}
				edges.Add(construct.SimpleEdge{Source: path[i-1], Target: res})

			}
		}
	}
	return edges, errs
}
