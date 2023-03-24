package graph

import (
	"github.com/dominikbraun/graph"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	ourFault = "This is a Klotho bug."
)

type (
	Directed[V Identifiable] struct {
		underlying graph.Graph[string, V]
	}

	Edge[V Identifiable] struct {
		Source      V
		Destination V
	}

	Identifiable interface {
		Id() string
	}
)

func NewDirected[V Identifiable]() *Directed[V] {
	return &Directed[V]{
		underlying: graph.New(V.Id, graph.Directed(), graph.Rooted()),
	}
}

func (d *Directed[V]) Roots() []V {
	// Note: this is inefficient. The graph library we use doesn't let us get just the roots, so we pull in
	// the full predecessor map, get all the ids with no outgoing edges, and then look up the vertex for each one
	// of those.
	// We can optimize later if needed.
	predecessors, err := d.underlying.PredecessorMap()
	if err != nil {
		// Very unexpected! This is only because the underlying graph store is generalized and supports returning err,
		// in case it's something like a SQL-backed store. Our store is in-memory and should never error out.
		panic(err)
	}
	var roots []V
	for vId, outgoing := range predecessors {
		if len(outgoing) == 0 {
			if v, err := d.underlying.Vertex(vId); err == nil {
				roots = append(roots, v)
			} else {
				zap.S().Errorf(`Couldn't resolve vertex with id="%s". %s`, vId, ourFault)
			}
		}
	}
	return roots
}

func (d *Directed[V]) OutgoingEdges(from V) []Edge[V] {
	return handleOutgoingEdges(d, from, func(destination V) Edge[V] {
		return Edge[V]{
			Source:      from,
			Destination: destination,
		}
	})
}

func (d *Directed[V]) IncomingEdges(to V) []Edge[V] {
	return handleIncomingEdges(d, to, func(destination V) Edge[V] {
		return Edge[V]{
			Source:      destination,
			Destination: to,
		}
	})
}

func (d *Directed[V]) AddVertex(v V) {
	err := d.underlying.AddVertex(v) // ignore errors if this is a duplicate
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		zap.S().With(zap.Error(err)).Errorf(`Unexpected error while adding %s. %s`, v, ourFault)
	}
}

func (d *Directed[V]) OutgoingVertices(from V) []V {
	return handleOutgoingEdges(d, from, func(destination V) V { return destination })
}

func (d *Directed[V]) IncomingVertices(to V) []V {
	return handleIncomingEdges(d, to, func(destination V) V { return destination })
}

func (d *Directed[V]) AddVerticesAndEdge(source V, dest V) {
	d.AddVertex(source)
	d.AddVertex(dest)
	err := d.underlying.AddEdge(source.Id(), dest.Id())
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while adding edge between "%v" and "%v"`, source, dest)
	}
}

func (d *Directed[V]) AddEdge(source string, dest string) {
	err := d.underlying.AddEdge(source, dest)
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while adding edge between "%v" and "%v"`, source, dest)
	}
}

func (d *Directed[V]) GetVertex(source string) V {
	v, err := d.underlying.Vertex(source)
	if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while getting vertex for "%v"`, source)
	}
	return v
}

func (d *Directed[V]) GetAllVertices() []V {
	predecessors, err := d.underlying.PredecessorMap()
	if err != nil {
		// Very unexpected! This is only because the underlying graph store is generalized and supports returning err,
		// in case it's something like a SQL-backed store. Our store is in-memory and should never error out.
		panic(err)
	}
	var vertices []V
	for vId := range predecessors {
		if v, err := d.underlying.Vertex(vId); err == nil {
			vertices = append(vertices, v)
		} else {
			zap.S().Errorf(`Couldn't resolve vertex with id="%s". %s`, vId, ourFault)
		}
	}
	return vertices
}

func (d *Directed[V]) GetEdge(source string, target string) graph.Edge[V] {
	v, err := d.underlying.Edge(source, target)
	if err != nil && !errors.Is(err, graph.ErrEdgeNotFound) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while getting vertex for "%v"`, source)
	}
	return v
}

func (d *Directed[V]) GetAllEdges() []Edge[V] {
	var results []Edge[V]

	fullAdjacency, err := d.underlying.AdjacencyMap()
	if err != nil {
		// Very unexpected! This is only because the underlying graph store is generalized and supports returning err,
		// in case it's something like a SQL-backed store. Our store is in-memory and should never error out.
		panic(err)
	}
	for _, edges := range fullAdjacency {
		for _, edge := range edges {
			sourceV, err := d.underlying.Vertex(edge.Source)
			if err != nil {
				zap.S().With(zap.Error(err)).Errorf(
					`Ignoring edge %v because I couldn't resolve the source vertex. %s`, edge, ourFault)
			}
			destV, err := d.underlying.Vertex(edge.Target)
			if err != nil {
				zap.S().With(zap.Error(err)).Errorf(
					`Ignoring edge %v because I couldn't resolve the destination vertex. %s`, edge, ourFault)
			}
			results = append(results, Edge[V]{Source: sourceV, Destination: destV})
		}
	}
	return results
}

func handleOutgoingEdges[V Identifiable, O any](d *Directed[V], from V, generate func(destination V) O) []O {
	// Note: this is very inefficient. The graph library we use doesn't let us get just the roots, so we pull in
	// the full predecessor map, get all the ids with no outgoing edges, and then look up the vertex for each one
	// of those.
	// This basically turns *each* edge traversal into an O(n) operation, where N is the size of the graph. That means
	// traversing the full graph is likely O(n²).
	// We can optimize later if needed.
	fullAdjacency, err := d.underlying.AdjacencyMap()
	if err != nil {
		// Very unexpected! This is only because the underlying graph store is generalized and supports returning err,
		// in case it's something like a SQL-backed store. Our store is in-memory and should never error out.
		panic(err)
	}
	var results []O
	vertexAdjacency, ok := fullAdjacency[from.Id()]
	if !ok {
		return results
	}
	for _, edge := range vertexAdjacency {
		if edge.Source != from.Id() {
			zap.S().Debugf(`Ignoring unexpected edge source from %v`, edge)
			continue
		}
		if toV, err := d.underlying.Vertex(edge.Target); err == nil {
			toAdd := generate(toV)
			results = append(results, toAdd)
		} else {
			zap.S().With(zap.Error(err)).Errorf(
				`Ignoring edge %v because I couldn't resolve the destination vertex. %s`, edge, ourFault)
		}
	}
	return results
}

func handleIncomingEdges[V Identifiable, O any](d *Directed[V], to V, generate func(destination V) O) []O {
	// Note: this is very inefficient. The graph library we use doesn't let us get just the roots, so we pull in
	// the full predecessor map, get all the ids with no outgoing edges, and then look up the vertex for each one
	// of those.
	// This basically turns *each* edge traversal into an O(n) operation, where N is the size of the graph. That means
	// traversing the full graph is likely O(n²).
	// We can optimize later if needed.
	fullAdjacency, err := d.underlying.AdjacencyMap()
	if err != nil {
		// Very unexpected! This is only because the underlying graph store is generalized and supports returning err,
		// in case it's something like a SQL-backed store. Our store is in-memory and should never error out.
		panic(err)
	}
	var results []O
	for _, v := range fullAdjacency {
		for _, edge := range v {
			if edge.Target != to.Id() {
				zap.S().Debugf(`Ignoring unexpected edge source from %v`, edge)
				continue
			}
			if toV, err := d.underlying.Vertex(edge.Source); err == nil {
				toAdd := generate(toV)
				results = append(results, toAdd)
			} else {
				zap.S().With(zap.Error(err)).Errorf(
					`Ignoring edge %v because I couldn't resolve the destination vertex. %s`, edge, ourFault)
			}
		}
	}
	return results
}
