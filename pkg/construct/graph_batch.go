package construct

import (
	"errors"
)

// GraphBatch can be used to batch adding vertices and edges to the graph,
// collecting errors in the [Err] field.
type GraphBatch struct {
	Graph
	Err error

	// errorAdding is to keep track on which resources we failed to add to the graph
	// so that we can ignore them when adding edges to not pollute the errors.
	errorAdding map[ResourceId]struct{}
}

func NewGraphBatch(g Graph) *GraphBatch {
	return &GraphBatch{
		Graph:       g,
		errorAdding: make(map[ResourceId]struct{}),
	}
}

func (b *GraphBatch) AddVertices(rs ...*Resource) {
	for _, r := range rs {
		err := b.Graph.AddVertex(r)
		if err == nil {
			continue
		}
		b.Err = errors.Join(b.Err, err)
		b.errorAdding[r.ID] = struct{}{}
	}
}

func (b *GraphBatch) AddEdges(es ...Edge) {
	for _, e := range es {
		if _, ok := b.errorAdding[e.Source]; ok {
			continue
		}
		if _, ok := b.errorAdding[e.Target]; ok {
			continue
		}

		err := b.Graph.AddEdge(e.Source, e.Target, CopyEdgeProps(e.Properties))
		b.Err = errors.Join(b.Err, err)
	}
}
