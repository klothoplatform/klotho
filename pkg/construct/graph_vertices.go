package construct

import (
	"errors"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
)

// TopologicalSort provides a stable topological ordering of resource IDs.
// This is a modified implementation of graph.StableTopologicalSort with the primary difference
// being any uses of the internal function `enqueueArbitrary`.
func TopologicalSort[T any](g graph.Graph[ResourceId, T]) ([]ResourceId, error) {
	return graph_addons.TopologicalSort(g, ResourceIdLess)
}

// ReverseTopologicalSort is like TopologicalSort, but returns the reverse order. This is primarily useful for
// IaC graphs to determine the order in which resources should be created.
func ReverseTopologicalSort[T any](g graph.Graph[ResourceId, T]) ([]ResourceId, error) {
	return graph_addons.ReverseTopologicalSort(g, ResourceIdLess)
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
