package path_selection

import (
	"errors"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

// determineCandidateWeight determines the weight of a candidate resource based on its relationship to the src and target resources
// and if it is already in the result graph.
//
// The weight is determined by the following:
// 1. If the candidate is downstream of the src or upstream of the target, add 10 to the weight
// 2. If the candidate is in the result graph, add 9 to the weight
// 3. if the candidate is existing determine how close it is to the src and target resources for additional weighting
//
// 'undirected' is from the 'ctx' raw view, but given as an argument here to avoid having to recompute it.
// 'desc' return is purely for debugging purposes, describing the weight calculation.
func determineCandidateWeight(
	ctx solution_context.SolutionContext,
	src, target construct.ResourceId,
	id construct.ResourceId,
	resultGraph construct.Graph,
	undirected construct.Graph,
) (weight int, errs error) {
	// note(gg) perf: these Downstream/Upstream functions don't need the full list and don't need to run twice
	downstreams, err := solution_context.Downstream(ctx, src, knowledgebase.ResourceDirectLayer)
	errs = errors.Join(errs, err)
	if collectionutil.Contains(downstreams, id) {
		weight += 10
	} else {
		downstreams, err := solution_context.Downstream(ctx, src, knowledgebase.ResourceGlueLayer)
		errs = errors.Join(errs, err)
		if collectionutil.Contains(downstreams, id) {
			weight += 5
		}
	}
	upstreams, err := solution_context.Upstream(ctx, target, knowledgebase.ResourceDirectLayer)
	errs = errors.Join(errs, err)
	if collectionutil.Contains(upstreams, id) {
		weight += 10
	} else {
		upstreams, err := solution_context.Upstream(ctx, target, knowledgebase.ResourceGlueLayer)
		errs = errors.Join(errs, err)
		if collectionutil.Contains(upstreams, id) {
			weight += 5
		}
	}

	// See if its currently in the result graph and if so add weight to increase chances of being reused
	_, err = resultGraph.Vertex(id)
	if err == nil {
		weight += 9
	}

	pather, err := construct.ShortestPaths(undirected, id, construct.DontSkipEdges)
	if err != nil {
		errs = errors.Join(errs, err)
		return
	}

	// We start at 8 so its weighted less than actually being upstream of the target or downstream of the src
	availableWeight := 10
	shortestPath, err := pather.ShortestPath(src)
	if err != nil {
		availableWeight = -5
	}
	for _, res := range shortestPath {
		if knowledgebase.GetFunctionality(ctx.KnowledgeBase(), res) != knowledgebase.Unknown {
			availableWeight -= 2
		} else {
			availableWeight -= 1
		}
	}

	shortestPath, err = pather.ShortestPath(target)
	if err != nil {
		// If we can't find a path to the src then we dont want to add divide by weight since its currently not reachable
		availableWeight = -5
	}
	for _, res := range shortestPath {
		if knowledgebase.GetFunctionality(ctx.KnowledgeBase(), res) != knowledgebase.Unknown {
			availableWeight -= 1
		}
	}

	// We make sure the divideWeightBy is at least 2 so that reusing resources is always valued higher than creating new ones if possible
	if availableWeight < 0 {
		availableWeight = 2
	}

	weight += availableWeight
	return
}

func BuildUndirectedGraph(g construct.Graph, kb knowledgebase.TemplateKB) (construct.Graph, error) {
	undirected := graph.NewWithStore(
		construct.ResourceHasher,
		graph_addons.NewMemoryStore[construct.ResourceId, *construct.Resource](),
		graph.Weighted(),
	)
	err := undirected.AddVerticesFrom(g)
	if err != nil {
		return nil, err
	}
	edges, err := g.Edges()
	if err != nil {
		return nil, err
	}
	for _, e := range edges {
		weight := 1
		// increase weights for edges that are connected to a functional resource
		if knowledgebase.GetFunctionality(kb, e.Source) != knowledgebase.Unknown {
			weight = 1000
		} else if knowledgebase.GetFunctionality(kb, e.Target) != knowledgebase.Unknown {
			weight = 1000
		}
		err := undirected.AddEdge(e.Source, e.Target, graph.EdgeWeight(weight))
		if err != nil {
			return nil, err
		}
	}
	return undirected, nil
}
