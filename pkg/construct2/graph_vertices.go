package construct2

import (
	"errors"
	"fmt"
	"sort"
)

// sortedIds is a helper type for sorting ResourceIds by purely their content, for use when deterministic ordering
// is desired (when no other sources of ordering are available).
type sortedIds []ResourceId

func (s sortedIds) Len() int {
	return len(s)
}

func ResourceIdLess(a, b ResourceId) bool {
	if a.Provider != b.Provider {
		return a.Provider < b.Provider
	}
	if a.Type != b.Type {
		return a.Type < b.Type
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	return a.Name < b.Name
}

func (s sortedIds) Less(i, j int) bool {
	return ResourceIdLess(s[i], s[j])
}

func (s sortedIds) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// ToplogicalSort provides a stable topological ordering of resource IDs.
// This is a modified implementation of graph.StableTopologicalSort with the primary difference
// being any uses of the internal function `enqueueArbitrary`.
func ToplogicalSort(g Graph) ([]ResourceId, error) {
	if !g.Traits().IsDirected {
		return nil, fmt.Errorf("topological sort cannot be computed on undirected graph")
	}

	predecessorMap, err := g.PredecessorMap()
	if err != nil {
		return nil, fmt.Errorf("failed to get predecessor map: %w", err)
	}

	if len(predecessorMap) == 0 {
		return nil, nil
	}

	queue := make([]ResourceId, 0)
	queued := make(map[ResourceId]struct{})
	enqueue := func(vs ...ResourceId) {
		for _, vertex := range vs {
			queue = append(queue, vertex)
			queued[vertex] = struct{}{}
		}
	}

	for vertex, predecessors := range predecessorMap {
		if len(predecessors) == 0 {
			enqueue(vertex)
		}
	}

	// enqueueArbitrary enqueues an arbitray but deterministic id from the remaining unvisited ids.
	// It should only be used if len(queue) == 0 && len(predecessorMap) > 0
	enqueueArbitrary := func() {
		remainingIds := make([]ResourceId, 0, len(predecessorMap))
		for vertex := range predecessorMap {
			remainingIds = append(remainingIds, vertex)
		}
		sort.Slice(remainingIds, func(i, j int) bool {
			// Pick an arbitrary vertex to start the queue based first on the number of remaining predecessors
			iPcount := len(predecessorMap[remainingIds[i]])
			jPcount := len(predecessorMap[remainingIds[j]])
			if iPcount != jPcount {
				return iPcount < jPcount
			}

			// Tie-break on the ID contents themselves
			return sortedIds(remainingIds).Less(i, j)
		})
		enqueue(remainingIds[0])
	}

	if len(queue) == 0 {
		enqueueArbitrary()
	}

	order := make([]ResourceId, 0, len(predecessorMap))
	visited := make(map[ResourceId]struct{})

	sort.Sort(sortedIds(queue))

	for len(queue) > 0 {
		currentVertex := queue[0]
		queue = queue[1:]

		if _, ok := visited[currentVertex]; ok {
			continue
		}

		order = append(order, currentVertex)
		visited[currentVertex] = struct{}{}
		delete(predecessorMap, currentVertex)

		frontier := make([]ResourceId, 0)

		for vertex, predecessors := range predecessorMap {
			delete(predecessors, currentVertex)

			if len(predecessors) != 0 {
				continue
			}

			if _, ok := queued[vertex]; ok {
				continue
			}

			frontier = append(frontier, vertex)
		}

		sort.Sort(sortedIds(frontier))

		enqueue(frontier...)

		if len(queue) == 0 && len(predecessorMap) > 0 {
			enqueueArbitrary()
		}
	}

	return order, nil
}

func reverseInplace[E any](a []E) {
	for i := 0; i < len(a)/2; i++ {
		a[i], a[len(a)-i-1] = a[len(a)-i-1], a[i]
	}
}

// ReverseTopologicalSort is like TopologicalSort, but returns the reverse order. This is primarily useful for
// IaC graphs to determine the order in which resources should be created.
func ReverseTopologicalSort(g Graph) ([]ResourceId, error) {
	topo, err := ToplogicalSort(g)
	if err != nil {
		return nil, err
	}
	reverseInplace(topo)
	return topo, nil
}

// WalkGraphFunc is much like `fs.WalkDirFunc` and is used in `WalkGraph` and `WalkGraphReverse` for the callback
// during graph traversal. Return `StopWalk` to end the walk.
type WalkGraphFunc func(id ResourceId, resource *Resource, nerr error) error

// StopWalk is a special error that can be returned from WalkGraphFunc to stop walking the graph.
// The resulting error from WalkGraph will be whatever was previously passed into the walk function.
var StopWalk = errors.New("stop walking")

func walkGraph(g Graph, ids []ResourceId, fn WalkGraphFunc) (err error) {
	for _, id := range ids {
		v, verr := g.Vertex(id)
		err = errors.Join(err, verr)
		err = fn(id, v, err)
	}
	return err
}

func WalkGraph(g Graph, fn WalkGraphFunc) error {
	topo, err := ToplogicalSort(g)
	if err != nil {
		return err
	}
	return walkGraph(g, topo, fn)
}

func WalkGraphReverse(g Graph, fn WalkGraphFunc) error {
	topo, err := ReverseTopologicalSort(g)
	if err != nil {
		return err
	}
	return walkGraph(g, topo, fn)
}
