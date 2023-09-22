package construct2

import (
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dominikbraun/graph"
)

type (
	Graph = graph.Graph[ResourceId, *Resource]
	Edge  = graph.Edge[ResourceId]
)

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

		targets := make([]ResourceId, 0, len(adjacent[id]))
		for t := range adjacent[id] {
			targets = append(targets, t)
		}
		sort.Sort(sortedIds(targets))

		for _, t := range targets {
			// Adjacent edges always have `id` as the source, so just write the target.
			_, err := fmt.Fprintf(w, "-> %s\n", t)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
