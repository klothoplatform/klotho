package construct2

import (
	"errors"

	"github.com/dominikbraun/graph"
)

func copyVertexProps(p graph.VertexProperties) func(*graph.VertexProperties) {
	return func(dst *graph.VertexProperties) {
		*dst = p
	}
}

func copyEdgeProps(p graph.EdgeProperties) func(*graph.EdgeProperties) {
	return func(dst *graph.EdgeProperties) {
		*dst = p
	}
}

// UpdateResourceId is used when a resource's ID changes. It updates the graph in-place, using the resource
// currently referenced by `old`. No-op if the resource ID hasn't changed.
func UpdateResourceId(g Graph, old ResourceId) error {
	r, props, err := g.VertexWithProperties(old)
	if err != nil {
		return err
	}
	// Short circuit if the resource ID hasn't changed.
	if old == r.ID {
		return nil
	}

	err = g.AddVertex(r, copyVertexProps(props))
	if err != nil {
		return err
	}

	adj, err := g.AdjacencyMap()
	if err != nil {
		return err
	}
	for _, edge := range adj[old] {
		err = errors.Join(
			err,
			g.AddEdge(r.ID, edge.Target, copyEdgeProps(edge.Properties)),
			g.RemoveEdge(edge.Source, edge.Target),
		)
	}
	if err != nil {
		return err
	}

	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}
	for _, edge := range pred[old] {
		err = errors.Join(
			err,
			g.AddEdge(edge.Source, r.ID, copyEdgeProps(edge.Properties)),
			g.RemoveEdge(edge.Source, edge.Target),
		)
	}
	if err != nil {
		return err
	}

	return g.RemoveVertex(old)
}
