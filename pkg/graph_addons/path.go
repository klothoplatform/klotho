package graph_addons

import (
	"fmt"

	"github.com/dominikbraun/graph"
)

type Path[K comparable] []K

func PathWeight[K comparable, V any](g graph.Graph[K, V], path Path[K]) (weight int, err error) {
	if !g.Traits().IsWeighted {
		return len(path), nil
	}
	for i := 1; i < len(path)-1; i++ {
		var e graph.Edge[V]
		e, err = g.Edge(path[i-1], path[i])
		if err != nil {
			err = fmt.Errorf("edge(path[%d], path[%d]): %w", i-1, i, err)
			return
		}
		weight += e.Properties.Weight
	}
	return
}

func (p Path[K]) Contains(k K) bool {
	for _, elem := range p {
		if elem == k {
			return true
		}
	}
	return false
}
