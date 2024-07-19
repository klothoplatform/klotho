package graph_addons

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/dominikbraun/graph"
)

// MemoryStore is like the default store returned by [graph.New] except that [AddVertex] and [AddEdge]
// are idempotent - they do not return an error if the vertex or edge already exists with the exact same value.
type MemoryStore[K comparable, T comparable] struct {
	lock             sync.RWMutex
	vertices         map[K]T
	vertexProperties map[K]graph.VertexProperties

	// outEdges and inEdges store all outgoing and ingoing edges for all vertices. For O(1) access,
	// these edges themselves are stored in maps whose keys are the hashes of the target vertices.
	outEdges map[K]map[K]graph.Edge[K] // source -> target
	inEdges  map[K]map[K]graph.Edge[K] // target -> source
}

type equaller interface {
	Equals(any) bool
}

func NewMemoryStore[K comparable, T comparable]() graph.Store[K, T] {
	return &MemoryStore[K, T]{
		vertices:         make(map[K]T),
		vertexProperties: make(map[K]graph.VertexProperties),
		outEdges:         make(map[K]map[K]graph.Edge[K]),
		inEdges:          make(map[K]map[K]graph.Edge[K]),
	}
}

func vertexPropsEqual(a, b graph.VertexProperties) bool {
	if a.Weight != b.Weight {
		return false
	}
	if len(a.Attributes) != len(b.Attributes) {
		return false
	}
	for k, aV := range a.Attributes {
		if bV, ok := b.Attributes[k]; !ok || aV != bV {
			return false
		}
	}
	return true
}

func (s *MemoryStore[K, T]) AddVertex(k K, t T, p graph.VertexProperties) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if p.Attributes == nil {
		p.Attributes = make(map[string]string)
	}

	if existing, ok := s.vertices[k]; ok {
		// Fastest check, use ==
		if t == existing && vertexPropsEqual(s.vertexProperties[k], p) {
			return nil
		}

		// Slower, check if it implements the equaller interface
		var t any = t // this is needed to satisfy the compiler, since Go can't type assert on a generic type
		if tEq, ok := t.(equaller); ok && tEq.Equals(existing) && vertexPropsEqual(s.vertexProperties[k], p) {
			return nil
		}

		return &graph.VertexAlreadyExistsError[K, T]{
			Key:           k,
			ExistingValue: existing,
		}
	}

	s.vertices[k] = t
	s.vertexProperties[k] = p

	return nil
}

func (s *MemoryStore[K, T]) ListVertices() ([]K, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	hashes := make([]K, 0, len(s.vertices))
	for k := range s.vertices {
		hashes = append(hashes, k)
	}

	return hashes, nil
}

func (s *MemoryStore[K, T]) VertexCount() (int, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.vertices), nil
}

func (s *MemoryStore[K, T]) Vertex(k K) (T, graph.VertexProperties, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.vertexWithLock(k)
}

// vertexWithLock returns the vertex and vertex properties - the caller must be holding at least a
// read-level lock.
func (s *MemoryStore[K, T]) vertexWithLock(k K) (T, graph.VertexProperties, error) {
	v, ok := s.vertices[k]
	if !ok {
		return v, graph.VertexProperties{}, &graph.VertexNotFoundError[K]{Key: k}
	}

	p := s.vertexProperties[k]

	return v, p, nil
}

func (s *MemoryStore[K, T]) RemoveVertex(k K) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.vertices[k]; !ok {
		return &graph.VertexNotFoundError[K]{Key: k}
	}

	count := 0
	if edges, ok := s.inEdges[k]; ok {
		inCount := len(edges)
		count += inCount
		if inCount == 0 {
			delete(s.inEdges, k)
		}
	}

	if edges, ok := s.outEdges[k]; ok {
		outCount := len(edges)
		count += outCount
		if outCount == 0 {
			delete(s.outEdges, k)
		}
	}

	if count > 0 {
		return &graph.VertexHasEdgesError[K]{Key: k, Count: count}
	}

	delete(s.vertices, k)
	delete(s.vertexProperties, k)

	return nil
}

func edgesEqual[K comparable](a, b graph.Edge[K]) bool {
	if a.Source != b.Source || a.Target != b.Target {
		return false
	}
	// Do all that fast/easy comparisons first so failures are quick
	if a.Properties.Weight != b.Properties.Weight {
		return false
	}
	if len(a.Properties.Attributes) != len(b.Properties.Attributes) {
		return false
	}
	for k, aV := range a.Properties.Attributes {
		if bV, ok := b.Properties.Attributes[k]; !ok || aV != bV {
			return false
		}
	}
	if a.Properties.Data == nil || b.Properties.Data == nil {
		// Can only safely check `==` if one is nil because a map cannot `==` anything else
		return a.Properties.Data == b.Properties.Data
	} else if aEq, ok := a.Properties.Data.(equaller); ok {
		return aEq.Equals(b.Properties.Data)
	} else if bEq, ok := b.Properties.Data.(equaller); ok {
		return bEq.Equals(a.Properties.Data)
	} else {
		// Do the reflection last, since that is slow. We need to use reflection unlike for attributes
		// because we don't know what type the data is.
		return reflect.DeepEqual(a.Properties.Data, b.Properties.Data)
	}
}

func (s *MemoryStore[K, T]) AddEdge(sourceHash, targetHash K, edge graph.Edge[K]) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, _, err := s.vertexWithLock(sourceHash); err != nil {
		return fmt.Errorf("could not get source vertex: %w", &graph.VertexNotFoundError[K]{Key: sourceHash})
	}
	if _, _, err := s.vertexWithLock(targetHash); err != nil {
		return fmt.Errorf("could not get target vertex: %w", &graph.VertexNotFoundError[K]{Key: targetHash})
	}

	if existing, ok := s.outEdges[sourceHash][targetHash]; ok {
		if !edgesEqual(existing, edge) {
			return &graph.EdgeAlreadyExistsError[K]{Source: sourceHash, Target: targetHash}
		}
	}

	if existing, ok := s.inEdges[targetHash][sourceHash]; ok {
		if !edgesEqual(existing, edge) {
			return &graph.EdgeAlreadyExistsError[K]{Source: sourceHash, Target: targetHash}
		}
	}

	if _, ok := s.outEdges[sourceHash]; !ok {
		s.outEdges[sourceHash] = make(map[K]graph.Edge[K])
	}
	s.outEdges[sourceHash][targetHash] = edge

	if _, ok := s.inEdges[targetHash]; !ok {
		s.inEdges[targetHash] = make(map[K]graph.Edge[K])
	}
	s.inEdges[targetHash][sourceHash] = edge

	return nil
}

func (s *MemoryStore[K, T]) UpdateEdge(sourceHash, targetHash K, edge graph.Edge[K]) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, err := s.edgeWithLock(sourceHash, targetHash); err != nil {
		return err
	}

	s.outEdges[sourceHash][targetHash] = edge
	s.inEdges[targetHash][sourceHash] = edge

	return nil
}

func (s *MemoryStore[K, T]) RemoveEdge(sourceHash, targetHash K) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.inEdges[targetHash], sourceHash)
	delete(s.outEdges[sourceHash], targetHash)
	return nil
}

func (s *MemoryStore[K, T]) Edge(sourceHash, targetHash K) (graph.Edge[K], error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.edgeWithLock(sourceHash, targetHash)
}

// edgeWithLock returns the edge - the caller must be holding at least a read-level lock.
func (s *MemoryStore[K, T]) edgeWithLock(sourceHash, targetHash K) (graph.Edge[K], error) {
	sourceEdges, ok := s.outEdges[sourceHash]
	if !ok {
		return graph.Edge[K]{}, &graph.EdgeNotFoundError[K]{Source: sourceHash, Target: targetHash}
	}

	edge, ok := sourceEdges[targetHash]
	if !ok {
		return graph.Edge[K]{}, &graph.EdgeNotFoundError[K]{Source: sourceHash, Target: targetHash}
	}

	return edge, nil
}

func (s *MemoryStore[K, T]) ListEdges() ([]graph.Edge[K], error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	res := make([]graph.Edge[K], 0)
	for _, edges := range s.outEdges {
		for _, edge := range edges {
			res = append(res, edge)
		}
	}
	return res, nil
}

// CreatesCycle is a fastpath version of [CreatesCycle] that avoids calling
// [PredecessorMap], which generates large amounts of garbage to collect.
//
// Because CreatesCycle doesn't need to modify the PredecessorMap, we can use
// inEdges instead to compute the same thing without creating any copies.
func (s *MemoryStore[K, T]) CreatesCycle(source, target K) (bool, error) {
	if source == target {
		return true, nil
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, _, err := s.vertexWithLock(source); err != nil {
		return false, fmt.Errorf("could not get source vertex: %w", err)
	}

	if _, _, err := s.vertexWithLock(target); err != nil {
		return false, fmt.Errorf("could not get target vertex: %w", err)
	}

	stack := []K{source}
	visited := make(map[K]struct{})

	var currentHash K
	for len(stack) > 0 {
		currentHash, stack = stack[len(stack)-1], stack[:len(stack)-1]

		if _, ok := visited[currentHash]; !ok {
			// If the adjacent vertex also is the target vertex, the target is a
			// parent of the source vertex. An edge would introduce a cycle.
			if currentHash == target {
				return true, nil
			}

			visited[currentHash] = struct{}{}

			for adjacency := range s.inEdges[currentHash] {
				stack = append(stack, adjacency)
			}
		}
	}

	return false, nil
}
