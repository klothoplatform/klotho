package path_selection

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

func ExpandEdge(
	ctx solution_context.SolutionContext,
	dep construct.ResourceEdge,
	tempGraph construct.Graph,
) (construct.Graph, error) {
	result := construct.NewGraph()
	var errs error
	errs = errors.Join(errs, runOnNamespaces(dep.Source, dep.Target, ctx, result))
	connected, err := connectThroughNamespace(dep.Source, dep.Target, ctx, result)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	if !connected {
		errs = errors.Join(errs, expandEdge(ctx, dep, tempGraph, result))
	}
	return result, errs
}

func expandEdge(
	ctx solution_context.SolutionContext,
	dep construct.ResourceEdge,
	tempGraph construct.Graph,
	g construct.Graph,
) error {
	paths, err := graph.AllPathsBetween(tempGraph, dep.Source.ID, dep.Target.ID)
	if err != nil {
		return err
	}
	var errs error
	// represents id to qualified type because we dont need to do that processing more than once
	for _, path := range paths {
		errs = errors.Join(errs, ExpandPath(ctx, dep, path, tempGraph, g))
	}
	path, err := graph.ShortestPath(tempGraph, dep.Source.ID, dep.Target.ID)
	if err != nil {
		return errors.Join(errs, fmt.Errorf("could not find shortest path between %s and %s: %w", dep.Source.ID, dep.Target.ID, err))
	}
	name := fmt.Sprintf("%s_%s", dep.Source.ID.Name, dep.Target.ID.Name)
	// rename phantom nodes
	result := make([]construct.ResourceId, len(path))
	for i, id := range path {
		if strings.HasPrefix(id.Name, PHANTOM_PREFIX) {
			id.Name = name
		}
		result[i] = id
	}
	return errors.Join(errs, addResourceListToGraph(ctx, g, result))
}

// ExpandEdge takes a given `selectedPath` and resolves it to a path of resourceIds that can be used
// for creating resources, or existing resources.
func ExpandPath(
	ctx solution_context.SolutionContext,
	dep construct.ResourceEdge,
	path []construct.ResourceId,
	g construct.Graph,
	resultGraph construct.Graph,
) error {
	if len(path) == 2 {
		return nil
	}

	type candidate struct {
		id             construct.ResourceId
		divideWeightBy int
	}

	var errs error

	nonBoundaryResources := path[1 : len(path)-1]

	// candidates maps the nonboundary index to the set of resources that could satisfy it
	// this is a helper to make adding all the edges to the graph easier.
	candidates := make([]map[construct.ResourceId]int, len(nonBoundaryResources))

	newResources := make(set.Set[construct.ResourceId])
	// Create new nodes for the path
	for i, node := range nonBoundaryResources {
		candidates[i] = make(map[construct.ResourceId]int)
		candidates[i][node] = 0
		newResources.Add(node)
	}
	if errs != nil {
		return errs
	}

	// NOTE(gg): if for some reason the path could contain a duplicated selector
	// this would just add the resource to the first match. I don't
	// think this should happen for a call into [ExpandEdge], but noting it just in case.
	matchesNonBoundary := func(id construct.ResourceId) int {
		for i, node := range nonBoundaryResources {
			typedNodeId := construct.ResourceId{Provider: node.Provider, Type: node.Type, Namespace: node.Namespace}
			if typedNodeId.Matches(id) {
				return i
			}
		}
		return -1
	}

	addCandidates := func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		matchIdx := matchesNonBoundary(id)
		if matchIdx < 0 {
			return nil
		}
		err := g.AddVertex(resource)
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			return errors.Join(nerr, err)
		}
		if _, ok := candidates[matchIdx][id]; !ok {
			candidates[matchIdx][id] = 0
		}
		downstreams, err := solution_context.Downstream(ctx, dep.Source.ID, knowledgebase.AllDepsLayer)
		nerr = errors.Join(nerr, err)
		if collectionutil.Contains(downstreams, id) {
			candidates[matchIdx][id] += 10
		}
		upstreams, err := solution_context.Upstream(ctx, dep.Target.ID, knowledgebase.AllDepsLayer)
		nerr = errors.Join(nerr, err)
		if collectionutil.Contains(upstreams, id) {
			candidates[matchIdx][id] += 10
		}

		// See if its currently in the result graph and if so add weight to increase chances of being reused
		_, err = resultGraph.Vertex(id)
		if err == nil {
			candidates[matchIdx][id] += 9
		}

		if candidates[matchIdx][id] >= 10 {
			return nerr
		}

		undirected, err := operational_rule.BuildUndirectedGraph(ctx)
		if err != nil {
			return errors.Join(nerr, err)
		}
		pather, err := construct.ShortestPaths(undirected, id, construct.DontSkipEdges)
		if err != nil {
			return err
		}

		// We start at 8 so its weighted less than actually being upstream of the target or downstream of the src
		availableWeight := 8
		shortestPath, err := pather.ShortestPath(dep.Source.ID)
		if err != nil {
			return errors.Join(nerr, err)
		}
		for _, res := range shortestPath {
			if ctx.KnowledgeBase().GetFunctionality(res) != knowledgebase.Unknown {
				availableWeight -= 1
			}
		}

		shortestPath, err = pather.ShortestPath(dep.Target.ID)
		if err != nil {
			return errors.Join(nerr, err)
		}
		for _, res := range shortestPath {
			if ctx.KnowledgeBase().GetFunctionality(res) != knowledgebase.Unknown {
				availableWeight -= 1
			}
		}

		// We make sure the divideWeightBy is at least 2 so that reusing resources is always valued higher than creating new ones if possible
		if availableWeight < 0 {
			availableWeight = 2
		}

		candidates[matchIdx][id] += availableWeight
		return nerr
	}
	// We need to add candidates which exist in our current result graph so we can reuse them. We do this in case
	// we have already performed expansions to ensure the namespaces are connected, etc
	construct.WalkGraph(resultGraph, func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		return addCandidates(id, resource, nerr)
	})

	// Add all other candidates which exist within the graph
	construct.WalkGraph(ctx.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		return addCandidates(id, resource, nerr)
	})

	predecessors, err := ctx.DataflowGraph().PredecessorMap()
	if err != nil {
		return err
	}

	adjacent, err := ctx.DataflowGraph().AdjacencyMap()
	if err != nil {
		return err
	}

	// addEdge checks whether the edge should be added according to the following rules:
	// 1. If it connects two new resources, always add it
	// 2. If the edge exists, and its template specifies it is unique, only add it if it's an existing edge
	// 3. Otherwise, add it
	addEdge := func(source, target candidate) {

		weight := calculateEdgeWeight(construct.SimpleEdge{Source: dep.Source.ID, Target: dep.Target.ID},
			source.id, target.id, ctx.KnowledgeBase())
		// These phantom resources should already exist in the graph

		if source.id != dep.Source.ID {
			if source.divideWeightBy != 0 {
				weight /= source.divideWeightBy
			}
		}
		if target.id != dep.Target.ID {
			if target.divideWeightBy != 0 {
				weight /= target.divideWeightBy
			}
		}

		tmpl := ctx.KnowledgeBase().GetEdgeTemplate(source.id, target.id)
		if tmpl == nil {
			errs = errors.Join(errs, fmt.Errorf("could not find edge template for %s -> %s", source.id, target.id))
			return
		}
		if tmpl.Unique.Target {
			pred := predecessors[target.id]
			for origSource := range pred {
				if tmpl.Source.Matches(origSource) && origSource != source.id {
					return
				}
			}
		}
		if tmpl.Unique.Source {
			adj := adjacent[source.id]
			for origTarget := range adj {
				if tmpl.Target.Matches(origTarget) && origTarget != target.id {
					return
				}
			}
		}

		err := g.AddEdge(source.id, target.id, graph.EdgeWeight(weight))
		if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) && !errors.Is(err, graph.ErrEdgeCreatesCycle) {
			errs = errors.Join(errs, err)
		}
	}

	for i, resCandidates := range candidates {
		for id, weight := range resCandidates {
			if i == 0 {
				addEdge(candidate{id: dep.Source.ID}, candidate{id: id, divideWeightBy: weight})
				continue
			}

			sources := candidates[i-1]

			for source, w := range sources {
				addEdge(candidate{id: source, divideWeightBy: w}, candidate{id: id, divideWeightBy: weight})
			}

		}
	}
	if len(candidates) > 0 {
		for c, weight := range candidates[len(candidates)-1] {
			addEdge(candidate{id: c, divideWeightBy: weight}, candidate{id: dep.Target.ID})
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func runOnNamespaces(src, target *construct.Resource, ctx solution_context.SolutionContext, g construct.Graph) error {
	if src.ID.Namespace != "" && target.ID.Namespace != "" {
		kb := ctx.KnowledgeBase()
		targetNamespaceResourceId, err := kb.GetResourcesNamespaceResource(target)
		if targetNamespaceResourceId.IsZero() {
			return fmt.Errorf("could not find namespace resource for %s", target)
		}
		if err != nil {
			return err
		}

		srcNamespaceResourceId, err := kb.GetResourcesNamespaceResource(src)
		if srcNamespaceResourceId.IsZero() {
			return fmt.Errorf("could not find namespace resource for %s", src)
		}
		if err != nil {
			return err
		}
		srcNamespaceResource, err := ctx.RawView().Vertex(srcNamespaceResourceId)
		if err != nil {
			return err
		}
		targetNamespaceResource, err := ctx.RawView().Vertex(targetNamespaceResourceId)
		if err != nil {
			return err
		}
		// if we have a namespace resource that is not the same as the target namespace resource
		tg, err := BuildPathSelectionGraph(construct.SimpleEdge{Source: srcNamespaceResourceId, Target: targetNamespaceResourceId}, kb, nil)
		if err != nil {
			return fmt.Errorf("could not build path selection graph: %w", err)
		}
		err = expandEdge(ctx, construct.ResourceEdge{Source: srcNamespaceResource, Target: targetNamespaceResource}, tg, g)
		if err != nil {
			return err
		}
	}
	return nil
}

func connectThroughNamespace(src, target *construct.Resource, ctx solution_context.SolutionContext, g construct.Graph) (
	connected bool,
	errs error,
) {
	kb := ctx.KnowledgeBase()
	targetNamespaceResource, _ := kb.GetResourcesNamespaceResource(target)
	if targetNamespaceResource.IsZero() {
		return
	}

	downstreams, err := solution_context.Downstream(ctx, src.ID, knowledgebase.ResourceLocalLayer)
	if err != nil {
		return connected, err
	}
	for _, downId := range downstreams {
		// Right now we only check for side effects of the same type
		// We may want to check for any side effects that could be namespaced into the target namespace since that would influence
		// the source resources connection to that target namespace resource
		if downId.QualifiedTypeName() != targetNamespaceResource.QualifiedTypeName() {
			continue
		}
		down, err := ctx.RawView().Vertex(downId)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		res, _ := kb.GetResourcesNamespaceResource(down)
		if res.IsZero() {
			continue
		}
		if res == targetNamespaceResource {
			continue
		}
		// if we have a namespace resource that is not the same as the target namespace resource
		tg, err := BuildPathSelectionGraph(construct.SimpleEdge{Source: res, Target: target.ID}, kb, nil)
		if err != nil {
			continue
		}
		err = expandEdge(ctx, construct.ResourceEdge{Source: down, Target: target}, tg, g)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		connected = true
	}

	return
}

func addResourceListToGraph(ctx solution_context.SolutionContext, g construct.Graph, resources []construct.ResourceId) error {
	for i, resource := range resources {
		r, err := ctx.RawView().Vertex(resource)
		if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
			continue
		}
		if r == nil {
			r = construct.CreateResource(resource)
		}
		_ = g.AddVertex(r)
		if i == 0 {
			continue
		}
		_ = g.AddEdge(resources[i-1], resource)
	}
	return nil
}
