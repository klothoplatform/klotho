package graph

import (
	"fmt"
	"sort"

	"github.com/dominikbraun/graph"
	"github.com/pkg/errors"
)

type (
	KvIterator[K comparable] interface {
		forEach(map[K]map[K]graph.Edge[K], func(K, map[K]graph.Edge[K]))
	}

	kvIteratorStable[K comparable] struct {
		// isLess is a function suitable for use in [sort.Slice].
		isLess func(K, K) bool
	}

	vertexAndNeighbors[K comparable] struct {
		key       K
		neighbors map[K]graph.Edge[K]
	}
)

var (
	stringIterator = kvIteratorStable[string]{
		isLess: func(k1 string, k2 string) bool {
			return k1 < k2
		},
	}
)

// StableTopologicalSort is a copy-paste of [graph.TopologicalSort], but with a stable ordering.
//
// There is no guarantee of the ordering, except that (a) it conforms to the general contract of [graph.TopologicalSort]
// and (b) that it is additionally stable across runs.
func StableTopologicalSort[K comparable, T any](g graph.Graph[K, T], iterator KvIterator[K]) ([]K, error) {
	if !g.Traits().IsDirected {
		return nil, fmt.Errorf("topological sort cannot be computed on undirected graph")
	}

	predecessorMap, err := g.PredecessorMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get predecessor map: %w", err)
	}

	queue := make([]K, 0)

	iterator.forEach(predecessorMap, func(vertex K, predecessors map[K]graph.Edge[K]) {
		if len(predecessors) == 0 {
			queue = append(queue, vertex)
		}
	})

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

		iterator.forEach(predecessorMap, func(vertex K, predecessors map[K]graph.Edge[K]) {
			delete(predecessors, currentVertex)

			if len(predecessors) == 0 {
				queue = append(queue, vertex)
			}
		})
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

func (iter kvIteratorStable[K]) forEach(in map[K]map[K]graph.Edge[K], f func(K, map[K]graph.Edge[K])) {
	asList := make([]vertexAndNeighbors[K], 0, len(in))
	for k, v := range in {
		asList = append(asList, vertexAndNeighbors[K]{key: k, neighbors: v})
	}
	sort.Slice(asList, func(i, j int) bool {
		return iter.isLess(asList[i].key, asList[j].key)
	})

	for _, kv := range asList {
		f(kv.key, kv.neighbors)
	}
}

// The following shows how we could implement an iterator with very low runtime cost -- hopefully low enough that we can
// propose this up to github.com/dominikbraun/graph
//
// type kvIteratorUnstable[K comparable] struct{}
//
// func (_ kvIteratorUnstable[K]) forEach(in map[K]map[K]graph.Edge[K], f func(K, map[K]graph.Edge[K])) {
// 	for k, v := range in {
// 		f(k, v)
// 	}
// }
