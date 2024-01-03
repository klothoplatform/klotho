package graph_addons

import "github.com/dominikbraun/graph"

func ReverseTopologicalSort[K comparable, T any](g graph.Graph[K, T], less func(K, K) bool) ([]K, error) {
	reverseLess := func(a, b K) bool {
		return !less(b, a)
	}
	topo, err := graph.StableTopologicalSort(g, reverseLess)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(topo)/2; i++ {
		topo[i], topo[len(topo)-i-1] = topo[len(topo)-i-1], topo[i]
	}
	return topo, nil
}
