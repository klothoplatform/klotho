package graph

import (
	"sort"

	"github.com/dominikbraun/graph"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	ourFault = "This is a Klotho bug."
)

type (
	Directed[V any] struct {
		underlying graph.Graph[string, V]
		hasher     func(V) string
	}

	Edge[V any] struct {
		Source      V
		Destination V
		Properties  graph.EdgeProperties
	}

	VertexProperties = graph.VertexProperties
	EdgeProperties   = graph.EdgeProperties
)

func NewDirected[V any](hasher func(V) string) *Directed[V] {

	return &Directed[V]{
		underlying: graph.New(hasher, graph.Directed(), graph.Rooted()),
		hasher:     hasher,
	}
}

func NewLike[V any](other *Directed[V]) *Directed[V] {
	return &Directed[V]{
		underlying: graph.NewLike(other.underlying),
		hasher:     other.hasher,
	}
}

func ToVertexAttributes(attributes map[string]string) func(*graph.VertexProperties) {
	return graph.VertexAttributes(attributes)
}

func AttributesFromVertexProperties(attributes graph.VertexProperties) map[string]string {
	return attributes.Attributes
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

func (d *Directed[V]) VertexIdsInTopologicalOrder() ([]string, error) {
	var iter KvIterator[string] = stringIterator
	return StableTopologicalSort(d.underlying, iter)
}

func (d *Directed[V]) ShortestPath(source, target string) ([]string, error) {
	path, err := graph.ShortestPath(d.underlying, source, target)
	if err != nil && errors.Is(err, graph.ErrTargetNotReachable) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return path, nil
}

func (d *Directed[V]) AllPaths(source, target string) ([][]string, error) {
	path, err := graph.AllPathsBetween(d.underlying, source, target)
	if err != nil && errors.Is(err, graph.ErrTargetNotReachable) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return path, nil
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

func (d *Directed[V]) RemoveVertex(v string) error {
	err := d.underlying.RemoveVertex(v)
	if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
		zap.S().With(zap.Error(err)).Errorf(`Unexpected error while adding %s. %s`, v, ourFault)
		return err
	} else if errors.Is(err, graph.ErrVertexNotFound) {
		zap.S().With(zap.Error(err)).Debugf(`Ignoring error while removing %s because it does not exist`, v)
	}
	return nil
}

func (d *Directed[V]) AddVertex(v V) {
	err := d.underlying.AddVertex(v) // ignore errors if this is a duplicate
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		zap.S().With(zap.Error(err)).Errorf(`Unexpected error while adding %s. %s`, v, ourFault)
	}
}

func (d *Directed[V]) AddVertexWithProperties(v V, options ...func(*graph.VertexProperties)) {
	err := d.underlying.AddVertex(v, options...) // ignore errors if this is a duplicate
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		zap.S().With(zap.Error(err)).Errorf(`Unexpected error while adding %s. %s`, v, ourFault)
	} else if err != nil && errors.Is(err, graph.ErrVertexAlreadyExists) {
		zap.S().With(zap.Error(err)).Debugf(`have to replace vertex since it already exists %s. %s`, v, ourFault)
		outgoingEdges := d.OutgoingEdges(v)
		for _, edge := range outgoingEdges {
			err := d.RemoveEdge(d.hasher(edge.Source), d.hasher(edge.Destination))
			if err != nil {
				zap.S().With(zap.Error(err)).Debugf(`error removing edge from %s. %s`, v, ourFault)
			}
		}
		incomingEdges := d.IncomingEdges(v)
		for _, edge := range incomingEdges {
			err := d.RemoveEdge(d.hasher(edge.Source), d.hasher(edge.Destination))
			if err != nil {
				zap.S().With(zap.Error(err)).Debugf(`error removing edge to %s. %s`, v, ourFault)
			}
		}
		err := d.RemoveVertex(d.hasher(v))
		if err != nil {
			zap.S().With(zap.Error(err)).Debugf(`error removing vertex %s. %s`, v, ourFault)
		}
		d.AddVertexWithProperties(v, options...)
		for _, edge := range outgoingEdges {
			d.AddEdge(d.hasher(edge.Source), d.hasher(edge.Destination), edge.Properties)
		}
		for _, edge := range incomingEdges {
			d.AddEdge(d.hasher(edge.Source), d.hasher(edge.Destination), edge.Properties)
		}
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

func (d *Directed[V]) GetVertexWithProperties(source string) (V, graph.VertexProperties) {
	v, props, err := d.underlying.VertexWithProperties(source)
	if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while getting vertex for "%v"`, source)
	}
	return v, props
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
	err := d.underlying.AddEdge(d.hasher(source), d.hasher(dest))
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while adding edge between "%v" and "%v"`, source, dest)
	}
}

func (d *Directed[V]) AddEdge(source string, dest string, data any) {
	err := d.underlying.AddEdge(source, dest, graph.EdgeData(data))
	if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while adding edge between "%v" and "%v"`, source, dest)
	} else if err != nil && errors.Is(err, graph.ErrEdgeAlreadyExists) && data != nil {
		zap.S().With("error", zap.Error(err)).Debugf(
			`Unexpected error while adding edge between "%v" and "%v". Replacing edge since new data was passed in`, source, dest)
		err = d.underlying.RemoveEdge(source, dest)
		if err != nil {
			zap.S().With("error", zap.Error(err)).Errorf(
				`Unexpected error while removing edge between "%v" and "%v". failed to replace edge`, source, dest)
		} else {
			d.AddEdge(source, dest, data)
		}
	}
}

func (d *Directed[V]) GetAllVertices() []V {
	predecessors, err := d.underlying.PredecessorMap()
	if err != nil {
		// Very unexpected! This is only because the underlying graph store is generalized and supports returning err,
		// in case it's something like a SQL-backed store. Our store is in-memory and should never error out.
		panic(err)
	}
	var vertices []V
	var ids []string
	for vId := range predecessors {
		if v, err := d.underlying.Vertex(vId); err == nil {
			ids = append(ids, d.hasher(v))
		} else {
			zap.S().Errorf(`Couldn't resolve vertex with id="%s". %s`, vId, ourFault)
		}
	}

	sort.Strings(ids)
	for _, id := range ids {
		vertices = append(vertices, d.GetVertex(id))
	}
	return vertices
}

func (d *Directed[V]) GetEdge(source string, target string) *Edge[V] {
	v, err := d.underlying.Edge(source, target)
	switch {
	case err == nil:
		return &Edge[V]{Source: v.Source, Destination: v.Target, Properties: v.Properties}

	case errors.Is(err, graph.ErrEdgeNotFound):
		return nil

	default:
		zap.S().With("error", zap.Error(err)).Errorf(
			`Unexpected error while getting vertex for "%v"`, source)
		return nil
	}
}

func (d *Directed[V]) RemoveEdge(source string, target string) error {
	return d.underlying.RemoveEdge(source, target)
}

func (d *Directed[V]) IdForNode(v V) string {
	return d.hasher(v)
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
			results = append(results, Edge[V]{Source: sourceV, Destination: destV, Properties: edge.Properties})
		}
	}
	return results
}

func (d *Directed[V]) CreatesCycle(source string, dest string) (bool, error) {
	return graph.CreatesCycle(d.underlying, source, dest)
}

func handleOutgoingEdges[V any, O any](d *Directed[V], from V, generate func(destination V) O) []O {
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
	vertexAdjacency, ok := fullAdjacency[d.hasher(from)]
	if !ok {
		return results
	}
	for _, edge := range vertexAdjacency {
		if edge.Source != d.hasher(from) {
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

func handleIncomingEdges[V any, O any](d *Directed[V], to V, generate func(destination V) O) []O {
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
			if edge.Target != d.hasher(to) {
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
