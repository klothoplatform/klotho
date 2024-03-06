package graph_addons

import (
	"sort"

	"github.com/dominikbraun/graph"
)

// TopologicalSort provides a stable topological ordering.
func TopologicalSort[K comparable, T any](g graph.Graph[K, T], less func(K, K) bool) ([]K, error) {
	predecessors, err := g.PredecessorMap()
	if err != nil {
		return nil, err
	}
	return topologicalSort(predecessors, less)
}

// topologicalSort performs a topological sort on a graph with the given dependencies.
// Whether the sort is regular or reverse is determined by whether the `deps` map is a PredecessorMap or AdjacencyMap.
// The `less` function is used to determine the order of vertices in the result.
// This is a modified implementation of graph.StableTopologicalSort with the primary difference
// being any uses of the internal function `enqueueArbitrary`.
func topologicalSort[K comparable](deps map[K]map[K]graph.Edge[K], less func(K, K) bool) ([]K, error) {
	if len(deps) == 0 {
		return nil, nil
	}

	queue := make([]K, 0)
	queued := make(map[K]struct{})
	enqueue := func(vs ...K) {
		for _, vertex := range vs {
			queue = append(queue, vertex)
			queued[vertex] = struct{}{}
		}
	}

	for vertex, vdeps := range deps {
		if len(vdeps) == 0 {
			enqueue(vertex)
		}
	}
	sort.Slice(queue, func(i, j int) bool {
		return less(queue[i], queue[j])
	})

	// enqueueArbitrary enqueues an arbitray but deterministic id from the remaining unvisited ids.
	// It should only be used if len(queue) == 0 && len(deps) > 0
	enqueueArbitrary := func() {
		remaining := make([]K, 0, len(deps))
		for vertex := range deps {
			remaining = append(remaining, vertex)
		}
		sort.Slice(remaining, func(i, j int) bool {
			// Start based first on the number of remaining deps, prioritizing vertices with fewer deps
			// to make it most likely to break any cycles, reducing the amount of arbitrary choices.
			ic := len(deps[remaining[i]])
			jc := len(deps[remaining[j]])
			if ic != jc {
				return ic < jc
			}

			// Tie-break using the less function on contents themselves
			return less(remaining[i], remaining[j])
		})
		enqueue(remaining[0])
	}

	if len(queue) == 0 {
		enqueueArbitrary()
	}

	order := make([]K, 0, len(deps))
	visited := make(map[K]struct{})

	for len(queue) > 0 {
		currentVertex := queue[0]
		queue = queue[1:]

		if _, ok := visited[currentVertex]; ok {
			continue
		}

		order = append(order, currentVertex)
		visited[currentVertex] = struct{}{}
		delete(deps, currentVertex)

		frontier := make([]K, 0)

		for vertex, predecessors := range deps {
			delete(predecessors, currentVertex)

			if len(predecessors) != 0 {
				continue
			}

			if _, ok := queued[vertex]; ok {
				continue
			}

			frontier = append(frontier, vertex)
		}

		sort.Slice(frontier, func(i, j int) bool {
			return less(frontier[i], frontier[j])
		})

		enqueue(frontier...)

		if len(queue) == 0 && len(deps) > 0 {
			enqueueArbitrary()
		}
	}

	return order, nil
}
