package path_selection

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

// determineCandidateWeight determines the weight of a candidate resource based on its relationship to the src and target resources
// and if it is already in the result graph
//
// The weight is determined by the following:
// 1. If the candidate is downstream of the src or upstream of the target, add 10 to the weight
// 2. If the candidate is in the result graph, add 9 to the weight
// 3. if the candidate is existing determine how close it is to the src and target resources for additional weighting
func determineCandidateWeight(
	ctx solution_context.SolutionContext,
	src, target construct.ResourceId,
	id construct.ResourceId,
	resultGraph construct.Graph,
) (weight int, errs error) {
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

	undirected, err := operational_rule.BuildUndirectedGraph(ctx)
	if err != nil {
		errs = errors.Join(errs, err)
		return
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
