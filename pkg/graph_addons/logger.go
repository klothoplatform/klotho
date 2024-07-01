package graph_addons

import (
	"github.com/dominikbraun/graph"
	"go.uber.org/zap"
)

type LoggingGraph[K comparable, T any] struct {
	graph.Graph[K, T]

	Log  *zap.SugaredLogger
	Hash func(T) K
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
		e, _ := g.Graph.Edge(sourceHash, targetHash)
		if e.Properties.Data == nil {
			g.Log.Debugf("AddEdge(%v -> %v)", sourceHash, targetHash)
		} else {
			g.Log.Debugf("AddEdge(%v -> %v, %+v)", sourceHash, targetHash, e.Properties.Data)
		}
	}
	return err
}

func (g LoggingGraph[K, T]) AddEdgesFrom(other graph.Graph[K, T]) error {
	// TODO
	return g.Graph.AddEdgesFrom(other)
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

func (g LoggingGraph[K, T]) Clone() (graph.Graph[K, T], error) {
	cloned, err := g.Graph.Clone()
	if err != nil {
		return nil, err
	}
	return LoggingGraph[K, T]{Log: g.Log, Graph: cloned}, nil
}
