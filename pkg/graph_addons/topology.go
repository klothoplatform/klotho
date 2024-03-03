package graph_addons

import (
	"fmt"
	"sort"

	"github.com/dominikbraun/graph"
)

// TopologicalSort provides a stable topological ordering of resource IDs.
// This is a modified implementation of graph.StableTopologicalSort with the primary difference
// being any uses of the internal function `enqueueArbitrary`.
func TopologicalSort[K comparable, T any](g graph.Graph[K, T], less func(K, K) bool) ([]K, error) {
	return toplogicalSort(g, less, false)
}

func toplogicalSort[K comparable, T any](g graph.Graph[K, T], less func(K, K) bool, invertLess bool) ([]K, error) {
	if !g.Traits().IsDirected {
		return nil, fmt.Errorf("topological sort cannot be computed on undirected graph")
	}

	predecessorMap, err := g.PredecessorMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get predecessor map: %w", err)
	}

	if len(predecessorMap) == 0 {
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

	for vertex, predecessors := range predecessorMap {
		if len(predecessors) == 0 {
			enqueue(vertex)
		}
	}

	// enqueueArbitrary enqueues an arbitray but deterministic id from the remaining unvisited ids.
	// It should only be used if len(queue) == 0 && len(predecessorMap) > 0
	enqueueArbitrary := func() {
		remainingIds := make([]K, 0, len(predecessorMap))
		for vertex := range predecessorMap {
			remainingIds = append(remainingIds, vertex)
		}
		sort.Slice(remainingIds, func(i, j int) bool {
			// Pick an arbitrary vertex to start the queue based first on the number of remaining predecessors
			iPcount := len(predecessorMap[remainingIds[i]])
			jPcount := len(predecessorMap[remainingIds[j]])
			if iPcount != jPcount {
				if invertLess {
					return iPcount >= jPcount
				} else {
					return iPcount < jPcount
				}
			}

			// Tie-break on the ID contents themselves
			if invertLess {
				return !less(remainingIds[i], remainingIds[j])
			}
			return less(remainingIds[i], remainingIds[j])
		})
		enqueue(remainingIds[0])
	}

	if len(queue) == 0 {
		enqueueArbitrary()
	}

	order := make([]K, 0, len(predecessorMap))
	visited := make(map[K]struct{})

	if invertLess {
		sort.Slice(queue, func(i, j int) bool {
			return !less(queue[i], queue[j])
		})
	} else {
		sort.Slice(queue, func(i, j int) bool {
			return less(queue[i], queue[j])
		})
	}

	for len(queue) > 0 {
		currentVertex := queue[0]
		queue = queue[1:]

		if _, ok := visited[currentVertex]; ok {
			continue
		}

		order = append(order, currentVertex)
		visited[currentVertex] = struct{}{}
		delete(predecessorMap, currentVertex)

		frontier := make([]K, 0)

		for vertex, predecessors := range predecessorMap {
			delete(predecessors, currentVertex)

			if len(predecessors) != 0 {
				continue
			}

			if _, ok := queued[vertex]; ok {
				continue
			}

			frontier = append(frontier, vertex)
		}

		if invertLess {
			sort.Slice(frontier, func(i, j int) bool {
				return !less(frontier[i], frontier[j])
			})
		} else {
			sort.Slice(frontier, func(i, j int) bool {
				return less(frontier[i], frontier[j])
			})
		}

		enqueue(frontier...)

		if len(queue) == 0 && len(predecessorMap) > 0 {
			enqueueArbitrary()
		}
	}

	return order, nil
}
