package graph_addons

import (
	"errors"

	"github.com/dominikbraun/graph"
)

func ReplaceVertex[K comparable, T any](g graph.Graph[K, T], oldId K, newValue T, hasher func(T) K) error {
	newKey := hasher(newValue)
	if newKey == oldId {
		return nil
	}

	_, props, err := g.VertexWithProperties(oldId)
	if err != nil {
		return err
	}

	err = g.AddVertex(newValue, func(vp *graph.VertexProperties) { *vp = props })
	if err != nil {
		return err
	}

	edges, err := g.Edges()
	if err != nil {
		return err
	}

	var errs error
	for _, e := range edges {
		if e.Source != oldId && e.Target != oldId {
			continue
		}

		newEdge := e
		if e.Source == oldId {
			newEdge.Source = newKey
		}
		if e.Target == oldId {
			newEdge.Target = newKey
		}
		errs = errors.Join(errs,
			g.RemoveEdge(e.Source, e.Target),
			g.AddEdge(newEdge.Source, newEdge.Target, func(ep *graph.EdgeProperties) { *ep = e.Properties }),
		)
	}
	if errs != nil {
		return errs
	}

	return g.RemoveVertex(oldId)
}
