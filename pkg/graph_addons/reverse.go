package graph_addons

import "github.com/dominikbraun/graph"

func ReverseTopologicalSort[K comparable, T any](g graph.Graph[K, T], less func(K, K) bool) ([]K, error) {
	return toplogicalSort(g, less, true)
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
