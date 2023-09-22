package construct2

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/dominikbraun/graph"
)

type Graph = graph.Graph[ResourceId, *Resource]

func NewGraph() Graph {
	return Graph(graph.New(
		func(r *Resource) ResourceId {
			return r.ID
		},
		graph.Directed(),
	))
}

func NewAcyclicGraph() Graph {
	return Graph(graph.New(
		func(r *Resource) ResourceId {
			return r.ID
		},
		graph.Directed(),
		graph.Acyclic(),
		graph.PreventCycles(),
	))
}

// isIdLess is used as the tie-breaker for when two resources have the same topological rank. It's arbitrary, but
// a stable ordering.
func isIdLess(a, b ResourceId) bool {
	return a.Provider < b.Provider || a.Type < b.Type || a.Namespace < b.Namespace || a.Name < b.Name
}

// ToplogicalSort provides a stable topological ordering of resource IDs.
func ToplogicalSort(g Graph) ([]ResourceId, error) {
	rids, err := graph.StableTopologicalSort(g, isIdLess)
	if err == nil {
		return rids, nil
	}
	if !strings.Contains(err.Error(), "cycles") {
		// kinda hacky, but we don't get a typed error for this case from the library
		return nil, err
	}
	// Remove cycles by converting to a spanning tree, then do the sort again.
	// This is a bit inefficient, so only do this if we couldn't get a topological sort
	// off of the original graph.
	mst, err := graph.MinimumSpanningTree(g)
	if err != nil {
		return nil, err
	}
	return graph.StableTopologicalSort(mst, isIdLess)
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

func Hash(g Graph) ([]byte, error) {
	sum := sha256.New()
	err := stringTo(g, sum)
	return sum.Sum(nil), err
}

func String(g Graph) (string, error) {
	w := new(strings.Builder)
	err := stringTo(g, w)
	return w.String(), err
}

func stringTo(g Graph, w io.Writer) error {
	topo, err := ToplogicalSort(g)
	if err != nil {
		return err
	}
	adjacent, err := g.AdjacencyMap()
	if err != nil {
		return err
	}

	for _, id := range topo {
		_, err := fmt.Fprintf(w, "%s\n", id)
		if err != nil {
			return err
		}

		for _, edge := range adjacent[id] {
			// Adjacent edges always have `id` as the source, so just write the target.
			_, err := fmt.Fprintf(w, "-> %s\n", edge.Target)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
