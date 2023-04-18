package graph

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/pkg/errors"
)

func StableTopologicalSort[K comparable, T any](g graph.Graph[K, T]) ([]K, error) {
	if !g.Traits().IsDirected {
		return nil, fmt.Errorf("topological sort cannot be computed on undirected graph")
	}

	predecessorMap, err := g.PredecessorMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get predecessor map: %w", err)
	}

	queue := make([]K, 0)

	for vertex, predecessors := range predecessorMap {
		if len(predecessors) == 0 {
			queue = append(queue, vertex)
		}
	}

	order := make([]K, 0, len(predecessorMap))
	visited := make(map[K]struct{})

	for len(queue) > 0 {
		currentVertex := queue[0]
		queue = queue[1:]

		if _, ok := visited[currentVertex]; ok {
			continue
		}

		order = append(order, currentVertex)
		visited[currentVertex] = struct{}{}

		for vertex, predecessors := range predecessorMap {
			delete(predecessors, currentVertex)

			if len(predecessors) == 0 {
				queue = append(queue, vertex)
			}
		}
	}

	gOrder, err := g.Order()
	if err != nil {
		return nil, fmt.Errorf("failed to get graph order: %w", err)
	}

	if len(order) != gOrder {
		return nil, errors.New("topological sort cannot be computed on graph with cycles")
	}

	return order, nil
}
