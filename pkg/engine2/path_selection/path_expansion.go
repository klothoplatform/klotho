package path_selection

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

// ExpandEdges
func ExpandEdge(
	ctx solution_context.SolutionContext,
	dep construct.ResourceEdge,
	validPath Path,
	edgeData EdgeData,
) ([]construct.ResourceId, error) {
	zap.S().Debugf("Expanding Edge for %s -> %s", dep.Source, dep.Target)

	g := construct.NewAcyclicGraph(graph.Weighted())
	var errs error

	// Create the known starting and ending nodes
	errs = errors.Join(errs, g.AddVertex(dep.Source))
	errs = errors.Join(errs, g.AddVertex(dep.Target))

	nonBoundaryResources := validPath.Nodes[1 : len(validPath.Nodes)-1]

	// candidates maps the nonboundary index to the set of resources that could satisfy it
	// this is a helper to make adding all the edges to the graph easier.
	candidates := make([]set.Set[construct.ResourceId], len(nonBoundaryResources))

	// Create new nodes for the path
	newResources := make(set.Set[construct.ResourceId])
	name := fmt.Sprintf("%s_%s", dep.Source.ID.Name, dep.Target.ID.Name)
	for i, node := range nonBoundaryResources {
		if node.Name == "" {
			node.Name = name
		}
		errs = errors.Join(errs, g.AddVertex(construct.CreateResource(node)))
		newResources.Add(node)
		candidates[i] = make(set.Set[construct.ResourceId])
		candidates[i].Add(node)
	}
	if errs != nil {
		return nil, errs
	}

	// NOTE: if for some reason the path could contain a duplicated selector
	// this would just add the resource to the first match. I (gordon) don't
	// think this should happen for a call into [ExpandEdge], but noting it just in case.
	matchesNonBoundary := func(id construct.ResourceId) int {
		for i, node := range nonBoundaryResources {
			if node.Matches(id) {
				return i
			}
		}
		return -1
	}

	// Add all the candidates from the graph based on what's downstream of the source or upstream of the target
	downstreams, err := solution_context.Downstream(ctx, dep.Source.ID, knowledgebase.AllDepsLayer)
	if err != nil {
		return nil, err
	}
	for _, downId := range downstreams {
		matchIdx := matchesNonBoundary(downId)
		if matchIdx < 0 {
			continue
		}
		down, err := ctx.RawView().Vertex(downId)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		errs = errors.Join(errs, construct.IgnoreExists(g.AddVertex(down)))
		candidates[matchIdx].Add(downId)
	}
	upstreams, err := solution_context.Upstream(ctx, dep.Target.ID, knowledgebase.AllDepsLayer)
	if err != nil {
		return nil, err
	}
	for _, upId := range upstreams {
		matchIdx := matchesNonBoundary(upId)
		if matchIdx < 0 {
			continue
		}
		up, err := ctx.RawView().Vertex(upId)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		errs = errors.Join(errs, construct.IgnoreExists(g.AddVertex(up)))
		candidates[matchIdx].Add(upId)
	}
	if errs != nil {
		return nil, errs
	}

	predecessors, err := ctx.DataflowGraph().PredecessorMap()
	if err != nil {
		return nil, err
	}

	adjacent, err := ctx.DataflowGraph().AdjacencyMap()
	if err != nil {
		return nil, err
	}

	// addEdge checks whether the edge should be added according to the following rules:
	// 1. If it connects two new resources, always add it
	// 2. If the edge exists, and its template specifies it is unique, only add it if it's an existing edge
	// 3. Otherwise, add it
	addEdge := func(source, target construct.ResourceId) {
		newSource := newResources.Contains(source)
		newTarget := newResources.Contains(target)
		if newSource && newTarget {
			// new edges get double weight to encourage using existing resources
			errs = errors.Join(errs, g.AddEdge(source, target, graph.EdgeWeight(2)))
			return
		}

		if !newSource && !newTarget {
			// edge already exists in the graph, just add it
			errs = errors.Join(errs, g.AddEdge(source, target, graph.EdgeWeight(1)))
			return
		}

		tmpl := ctx.KnowledgeBase().GetEdgeTemplate(source, target)
		if tmpl == nil {
			errs = errors.Join(errs, fmt.Errorf("could not find edge template for %s -> %s", source, target))
			return
		}
		if newSource && tmpl.Unique.Source {
			pred := predecessors[target]
			for origSource := range pred {
				if tmpl.Source.Matches(origSource) {
					return
				}
			}
		}
		if newTarget && tmpl.Unique.Target {
			adj := adjacent[source]
			for origTarget := range adj {
				if tmpl.Target.Matches(origTarget) {
					return
				}
			}
		}
		errs = errors.Join(errs, g.AddEdge(source, target, graph.EdgeWeight(1)))
	}

	for i, resCandidates := range candidates {
		for candidate := range resCandidates {
			if i == 0 {
				addEdge(dep.Source.ID, candidate)
				continue
			}

			isNew := newResources.Contains(candidate)
			sources := candidates[i-1]

			if isNew {
				for source := range sources {
					addEdge(source, candidate)
				}
			} else {
				for pred := range predecessors[candidate] {
					if sources.Contains(pred) {
						addEdge(pred, candidate)
					}
				}
			}
		}
	}
	for candidate := range candidates[len(candidates)-1] {
		addEdge(candidate, dep.Target.ID)
	}
	if errs != nil {
		return nil, errs
	}

	return graph.ShortestPath(g, dep.Source.ID, dep.Target.ID)
}
