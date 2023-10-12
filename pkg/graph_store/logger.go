package graph_store

import (
	"github.com/dominikbraun/graph"
	"go.uber.org/zap"
)

type LoggingGraph[K comparable, T any] struct {
	Log   *zap.SugaredLogger
	Graph graph.Graph[K, T]
	Hash  func(T) K
}

func (g LoggingGraph[K, T]) Traits() *graph.Traits {
	return g.Graph.Traits()
}

func (g LoggingGraph[K, T]) AddVertex(value T, options ...func(*graph.VertexProperties)) error {
	err := g.Graph.AddVertex(value, options...)
	if err != nil {
		g.Log.Errorf("AddVertex(%v) error: %v", g.Hash(value), err)
	} else {
		g.Log.Debugf("AddVertex(%v)", g.Hash(value))
	}
	return err
}

func (g LoggingGraph[K, T]) AddVerticesFrom(other graph.Graph[K, T]) error {
	// TODO
	return g.Graph.AddVerticesFrom(other)
}

func (g LoggingGraph[K, T]) Vertex(hash K) (T, error) {
	return g.Graph.Vertex(hash)
}

func (g LoggingGraph[K, T]) VertexWithProperties(hash K) (T, graph.VertexProperties, error) {
	return g.Graph.VertexWithProperties(hash)
}

func (g LoggingGraph[K, T]) RemoveVertex(hash K) error {
	err := g.Graph.RemoveVertex(hash)
	if err != nil {
		g.Log.Errorf("RemoveVertex(%v) error: %v", hash, err)
	} else {
		g.Log.Debugf("RemoveVertex(%v)", hash)
	}
	return err
}

func (g LoggingGraph[K, T]) AddEdge(sourceHash K, targetHash K, options ...func(*graph.EdgeProperties)) error {
	err := g.Graph.AddEdge(sourceHash, targetHash, options...)
	if err != nil {
		g.Log.Errorf("AddEdge(%v -> %v) error: %v", sourceHash, targetHash, err)
	} else {
		g.Log.Debugf("AddEdge(%v -> %v)", sourceHash, targetHash)
	}
	return err
}

func (g LoggingGraph[K, T]) AddEdgesFrom(other graph.Graph[K, T]) error {
	// TODO
	return g.Graph.AddEdgesFrom(other)
}

func (g LoggingGraph[K, T]) Edge(sourceHash K, targetHash K) (graph.Edge[T], error) {
	return g.Graph.Edge(sourceHash, targetHash)
}

func (g LoggingGraph[K, T]) Edges() ([]graph.Edge[K], error) {
	return g.Graph.Edges()
}

func (g LoggingGraph[K, T]) UpdateEdge(source K, target K, options ...func(properties *graph.EdgeProperties)) error {
	err := g.Graph.UpdateEdge(source, target, options...)
	if err != nil {
		g.Log.Errorf("UpdateEdge(%v, %v) error: %v", source, target, err)
	} else {
		g.Log.Debugf("UpdateEdge(%v, %v)", source, target)
	}
	return err
}

func (g LoggingGraph[K, T]) RemoveEdge(source K, target K) error {
	err := g.Graph.RemoveEdge(source, target)
	if err != nil {
		g.Log.Errorf("RemoveEdge(%v, %v) error: %v", source, target, err)
	} else {
		g.Log.Debugf("RemoveEdge(%v, %v)", source, target)
	}
	return err
}

func (g LoggingGraph[K, T]) AdjacencyMap() (map[K]map[K]graph.Edge[K], error) {
	return g.Graph.AdjacencyMap()
}

func (g LoggingGraph[K, T]) PredecessorMap() (map[K]map[K]graph.Edge[K], error) {
	return g.Graph.PredecessorMap()
}

func (g LoggingGraph[K, T]) Clone() (graph.Graph[K, T], error) {
	cloned, err := g.Graph.Clone()
	if err != nil {
		return nil, err
	}
	return LoggingGraph[K, T]{Log: g.Log, Graph: cloned}, nil
}

func (g LoggingGraph[K, T]) Order() (int, error) {
	return g.Graph.Order()
}

func (g LoggingGraph[K, T]) Size() (int, error) {
	return g.Graph.Size()
}
