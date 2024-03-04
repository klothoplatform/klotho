package graph_addons

import "github.com/dominikbraun/graph"

// ReverseLess is a helper function that returns a new less function that reverses the order of the original less function.
func ReverseLess[K any](less func(K, K) bool) func(K, K) bool {
	return func(a, b K) bool {
		return less(b, a)
	}
}

// TopologicalSort provides a stable topological ordering.
func ReverseTopologicalSort[K comparable, T any](g graph.Graph[K, T], less func(K, K) bool) ([]K, error) {
	adjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	return topologicalSort(adjacencyMap, less)
}

func ReverseGraph[K comparable, T any](g graph.Graph[K, T]) (graph.Graph[K, T], error) {
	reverse := graph.NewLike(g)
	err := reverse.AddVerticesFrom(g)
	if err != nil {
		return nil, err
	}
	edges, err := g.Edges()
	if err != nil {
		return nil, err
	}
	for _, e := range edges {
		err = reverse.AddEdge(e.Target, e.Source, func(ep *graph.EdgeProperties) {
			*ep = e.Properties
		})
		if err != nil {
			return nil, err
		}
	}
	return reverse, nil
}
