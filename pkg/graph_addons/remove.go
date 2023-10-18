package graph_addons

import (
	"errors"

	"github.com/dominikbraun/graph"
)

func RemoveVertexAndEdges[K comparable, T any](g graph.Graph[K, T], id K) error {
	edges, err := g.Edges()
	if err != nil {
		return err
	}

	var errs error
	for _, e := range edges {
		if e.Source != id && e.Target != id {
			continue
		}
		errs = errors.Join(errs, g.RemoveEdge(e.Source, e.Target))
	}
	if errs != nil {
		return errs
	}

	return g.RemoveVertex(id)
}
