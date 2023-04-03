package graph

import "sort"

// SortEdges sorts the edges in place by (source.Id(), dest.Id()). You can use this to normalize a collection of edges.
// This can be useful in tests. This method returns the slice you pass in, as a convenience.
func SortEdges[V Identifiable](edges []Edge[V]) []Edge[V] {
	return sortEdgesBy(edges, V.Id)
}

func SortEdgeIds(edges []Edge[string]) []Edge[string] {
	return sortEdgesBy(edges, func(s string) string {
		return s
	})
}

// EdgeIds converts a slice of edges into a slice of their ids.
func EdgeIds[V Identifiable](edges []Edge[V]) []Edge[string] {
	if edges == nil {
		return nil
	}
	result := make([]Edge[string], len(edges), len(edges))
	for i, edge := range edges {
		result[i] = Edge[string]{
			Source:      edge.Source.Id(),
			Destination: edge.Destination.Id(),
		}
	}
	return result
}

// VertexIds returns a set of all V.Id()s within the given slice of resources. Any ID collisions are silently ignored.
// The result is always a non-nil map, even if the incoming slice is nil.
func VertexIds[V Identifiable](vertices []V) map[string]struct{} {
	result := make(map[string]struct{}, len(vertices))
	for _, v := range vertices {
		result[v.Id()] = struct{}{}
	}
	return result
}

func sortEdgesBy[V any](edges []Edge[V], keyF func(V) string) []Edge[V] {
	sort.Slice(edges, func(i, j int) bool {
		iV, jV := edges[i], edges[j]
		return keyF(iV.Source) < keyF(jV.Source) && keyF(iV.Destination) < keyF(jV.Destination)
	})
	return edges
}
