package dag

import (
	"github.com/heimdalr/dag"
	"go.uber.org/zap"
)

type (
	Dag[V Identifiable] struct {
		underlying *dag.DAG
	}

	Edge[V Identifiable] struct {
		Source      V
		Destination V
	}

	Identifiable interface {
		Id() string
	}
)

func NewDag[V Identifiable]() *Dag[V] {
	return &Dag[V]{
		underlying: dag.NewDAG(),
	}
}

func (d *Dag[V]) Roots() []V {
	rootIds := d.underlying.GetRoots()
	roots := make([]V, 0, len(rootIds))
	for _, v := range rootIds {
		roots = append(roots, v.(V))
	}
	return roots
}

func (d *Dag[V]) OutgoingEdges(from V) []Edge[V] {
	return handleEdges(d, from, func(destination V) Edge[V] {
		return Edge[V]{
			Source:      from,
			Destination: destination,
		}
	})
}

func (d *Dag[V]) AddVertex(v V) {
	if err := d.underlying.AddVertexByID(v.Id(), v); err != nil {
		zap.S().With("error", err).Warnf(`Ignoring duplicate vertex "%v"`, v)
	}
}

func (d *Dag[V]) OutgoingVertices(from V) []V {
	return handleEdges(d, from, func(destination V) V { return destination })
}

func (d *Dag[V]) AddEdge(source V, dest V) {
	err := d.underlying.AddEdge(source.Id(), dest.Id())
	if err == nil {
		return
	}
	switch err.(type) {
	case dag.SrcDstEqualError:
		zap.S().Warnf(`Ignoring self-referential vertex "%v"`, source)
	case dag.EdgeDuplicateError:
		zap.S().Warnf(`Ignoring duplicate edge from "%v" to "%v"`, source, dest)
	case dag.EdgeLoopError:
		zap.S().With("error", zap.Error(err)).Errorf(`Ignoring edge from "%v" to "%v" because it would introduce a cyclic reference`, source, dest)
	default:
		zap.S().With("error", zap.Error(err)).Errorf(`Ignoring edge from "%v" to "%v" due to unknown error`, source, dest)
	}
}

func handleEdges[V Identifiable, O any](d *Dag[V], from V, generate func(destination V) O) []O {
	children, err := d.underlying.GetChildren(from.Id())
	if err != nil {
		zap.S().Warnf("Unknown vertex: %v", from)
		return []O{}
	}
	result := make([]O, 0, len(children))
	childrenMap, err := d.underlying.GetChildren(from.Id())
	if err != nil {
		zap.S().Errorf("Ignoring unknown vertex: %v", from)
		return []O{}
	}
	for _, child := range childrenMap {
		childV := child.(V)
		output := generate(childV)
		result = append(result, output)
	}
	return result
}
