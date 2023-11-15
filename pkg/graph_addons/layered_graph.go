package graph_addons

import (
	"errors"

	"github.com/dominikbraun/graph"
)

// LayeredGraph is a graph that is composed of multiple layers.
// When a vertex is added, it is added to the first layer (`[0]`).
// When an edge is added, if the source and target exist in the same layer, the edge is added to that layer,
// otherwise, the source and target are added to the first layer, and the edge is added.
// Remove and update operations are applied to all layers.
type LayeredGraph[K comparable, T any] []graph.Graph[K, T]

func (g LayeredGraph[K, T]) Traits() *graph.Traits {
	t := g[0].Traits()
	for i := 1; i < len(g); i++ {
		lt := g[i].Traits()
		t.IsDirected = t.IsDirected || lt.IsDirected
		t.IsAcyclic = t.IsAcyclic || lt.IsAcyclic
		t.IsWeighted = t.IsWeighted || lt.IsWeighted
		t.IsRooted = t.IsRooted || lt.IsRooted
		t.PreventCycles = t.PreventCycles || lt.PreventCycles
	}
	return t
}

func (g LayeredGraph[K, T]) AddVertex(value T, options ...func(*graph.VertexProperties)) error {
	return g[0].AddVertex(value, options...)
}

func (g LayeredGraph[K, T]) AddVerticesFrom(o graph.Graph[K, T]) error {
	return g[0].AddVerticesFrom(o)
}

func (g LayeredGraph[K, T]) Vertex(hash K) (v T, err error) {
	for _, layer := range g {
		if v, err = layer.Vertex(hash); err == nil {
			return v, nil
		} else if !errors.Is(err, graph.ErrVertexNotFound) {
			return
		}
	}
	err = graph.ErrVertexNotFound
	return
}

func (g LayeredGraph[K, T]) VertexWithProperties(hash K) (v T, p graph.VertexProperties, err error) {
	for _, layer := range g {
		if v, p, err = layer.VertexWithProperties(hash); err == nil {
			return v, p, nil
		} else if !errors.Is(err, graph.ErrVertexNotFound) {
			return
		}
	}
	err = graph.ErrVertexNotFound
	return
}
func (g LayeredGraph[K, T]) RemoveVertex(hash K) error {
	for _, layer := range g {
		err := layer.RemoveVertex(hash)
		if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
			return err
		}
	}
	return nil
}

func (g LayeredGraph[K, T]) AddEdge(sourceHash, targetHash K, options ...func(*graph.EdgeProperties)) error {
	var src, tgt T
	srcLayer, tgtLayer := -1, -1
	for i, layer := range g {
		sV, srcErr := layer.Vertex(sourceHash)
		tV, tgtErr := layer.Vertex(targetHash)
		if srcErr == nil && tgtErr == nil {
			return layer.AddEdge(sourceHash, targetHash, options...)
		}
		if srcErr == nil && srcLayer == -1 {
			srcLayer = i
			src = sV
		}
		if errors.Is(srcErr, graph.ErrVertexNotFound) {
			srcErr = nil
		}
		if tgtErr == nil && tgtLayer == -1 {
			tgtLayer = i
			tgt = tV
		}
		if errors.Is(tgtErr, graph.ErrVertexNotFound) {
			tgtErr = nil
		}
		err := errors.Join(srcErr, tgtErr)
		if err != nil {
			return err
		}
	}
	// no layer has both vertices, so add them both to the first layer
	// then add the edge
	err := errors.Join(
		g[0].AddVertex(src),
		g[0].AddVertex(tgt),
	)
	if err != nil {
		return err
	}
	return g[0].AddEdge(sourceHash, targetHash, options...)
}

func (g LayeredGraph[K, T]) AddEdgesFrom(o graph.Graph[K, T]) error {
	edges, err := o.Edges()
	if err != nil {
		return err
	}
	for _, edge := range edges {
		err = errors.Join(err, g.AddEdge(edge.Source, edge.Target, func(ep *graph.EdgeProperties) {
			*ep = edge.Properties
		}))
	}
	return err
}

func (g LayeredGraph[K, T]) Edge(sourceHash, targetHash K) (graph.Edge[T], error) {
	for _, layer := range g {
		e, err := layer.Edge(sourceHash, targetHash)
		if err == nil {
			return e, nil
		} else if !errors.Is(err, graph.ErrEdgeNotFound) {
			return graph.Edge[T]{}, err
		}
	}
	return graph.Edge[T]{}, graph.ErrEdgeNotFound
}

// Edges may return duplicate edges if an edge exists in multiple layers. This is intentional because those edges
// may contain different properties.
func (g LayeredGraph[K, T]) Edges() ([]graph.Edge[K], error) {
	var edges []graph.Edge[K]
	for _, layer := range g {
		layerEdges, err := layer.Edges()
		if err != nil {
			return nil, err
		}
		edges = append(edges, layerEdges...)
	}
	return edges, nil
}

func (g LayeredGraph[K, T]) UpdateEdge(source, target K, options ...func(properties *graph.EdgeProperties)) error {
	for _, layer := range g {
		err := layer.UpdateEdge(source, target, options...)
		if err != nil && !errors.Is(err, graph.ErrEdgeNotFound) {
			return err
		}
	}
	return nil
}

func (g LayeredGraph[K, T]) RemoveEdge(source, target K) error {
	for _, layer := range g {
		err := layer.RemoveEdge(source, target)
		if err != nil && !errors.Is(err, graph.ErrEdgeNotFound) {
			return err
		}
	}
	return nil
}

func (g LayeredGraph[K, T]) AdjacencyMap() (map[K]map[K]graph.Edge[K], error) {
	m := make(map[K]map[K]graph.Edge[K])
	// iterate backwards so that the first layer has the highest priority
	for _, layer := range g {
		adj, err := layer.AdjacencyMap()
		if err != nil {
			return nil, err
		}
		for s, ts := range adj {
			if m[s] == nil {
				m[s] = make(map[K]graph.Edge[K])
			}
			for t, e := range ts {
				existing, hasExisting := m[s][t]
				if hasExisting && existing.Properties.Weight > e.Properties.Weight {
					continue
				}
				m[s][t] = e
			}
		}
	}
	return m, nil
}

func (g LayeredGraph[K, T]) PredecessorMap() (map[K]map[K]graph.Edge[K], error) {
	m := make(map[K]map[K]graph.Edge[K])
	for _, layer := range g {
		pred, err := layer.PredecessorMap()
		if err != nil {
			return nil, err
		}
		for t, ss := range pred {
			if m[t] == nil {
				m[t] = make(map[K]graph.Edge[K])
			}
			for s, e := range ss {
				existing, hasExisting := m[t][s]
				if hasExisting && existing.Properties.Weight > e.Properties.Weight {
					continue
				}
				m[t][s] = e
			}
		}
	}
	return m, nil
}

func (g LayeredGraph[K, T]) Clone() (graph.Graph[K, T], error) {
	g2 := make(LayeredGraph[K, T], len(g))
	for i, layer := range g {
		var err error
		g2[i], err = layer.Clone()
		if err != nil {
			return nil, err
		}
	}
	return g2, nil
}

func (g LayeredGraph[K, T]) Order() (int, error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return 0, err
	}
	return len(adj), nil
}

func (g LayeredGraph[K, T]) Size() (int, error) {
	srcToTgt := make(map[K]K)
	edges, err := g.Edges()
	if err != nil {
		return 0, err
	}
	for _, edge := range edges {
		srcToTgt[edge.Source] = edge.Target
	}
	return len(srcToTgt), nil
}
