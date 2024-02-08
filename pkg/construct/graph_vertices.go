package construct

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/dominikbraun/graph"
)

// TopologicalSort provides a stable topological ordering of resource IDs.
// This is a modified implementation of graph.StableTopologicalSort with the primary difference
// being any uses of the internal function `enqueueArbitrary`.
func TopologicalSort[T any](g graph.Graph[ResourceId, T]) ([]ResourceId, error) {
	return toplogicalSort(g, false)
}

// ReverseTopologicalSort is like TopologicalSort, but returns the reverse order. This is primarily useful for
// IaC graphs to determine the order in which resources should be created.
func ReverseTopologicalSort[T any](g graph.Graph[ResourceId, T]) ([]ResourceId, error) {
	topo, err := toplogicalSort(g, true)
	if err != nil {
		return nil, err
	}
	slices.Reverse(topo)
	return topo, nil
}

func toplogicalSort[T any](g graph.Graph[ResourceId, T], invertLess bool) ([]ResourceId, error) {
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
				if invertLess {
					return iPcount >= jPcount
				} else {
					return iPcount < jPcount
				}
			}

			// Tie-break on the ID contents themselves
			if invertLess {
				return !SortedIds(remainingIds).Less(i, j)
			}
			return SortedIds(remainingIds).Less(i, j)
		})
		enqueue(remainingIds[0])
	}

	if len(queue) == 0 {
		enqueueArbitrary()
	}

	order := make([]ResourceId, 0, len(predecessorMap))
	visited := make(map[ResourceId]struct{})

	sort.Sort(SortedIds(queue))

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

		if invertLess {
			sort.Sort(sort.Reverse(SortedIds(frontier)))
		} else {
			sort.Sort(SortedIds(frontier))
		}

		enqueue(frontier...)

		if len(queue) == 0 && len(predecessorMap) > 0 {
			enqueueArbitrary()
		}
	}

	return order, nil
}

// WalkGraphFunc is much like `fs.WalkDirFunc` and is used in `WalkGraph` and `WalkGraphReverse` for the callback
// during graph traversal. Return `StopWalk` to end the walk.
type WalkGraphFunc func(id ResourceId, resource *Resource, nerr error) error

// StopWalk is a special error that can be returned from WalkGraphFunc to stop walking the graph.
// The resulting error from WalkGraph will be whatever was previously passed into the walk function.
var StopWalk = errors.New("stop walking")

func walkGraph(g Graph, ids []ResourceId, fn WalkGraphFunc) (nerr error) {
	for _, id := range ids {
		v, verr := g.Vertex(id)
		if verr != nil {
			return verr
		}
		err := fn(id, v, nerr)
		if errors.Is(err, StopWalk) {
			return
		}
		nerr = err
	}
	return
}

func WalkGraph(g Graph, fn WalkGraphFunc) error {
	topo, err := TopologicalSort(g)
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
