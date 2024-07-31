package construct

import (
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
)

type (
	Graph        = graph.Graph[ResourceId, *Resource]
	Edge         = graph.Edge[ResourceId]
	ResourceEdge = graph.Edge[*Resource]
)

func NewGraphWithOptions(options ...func(*graph.Traits)) Graph {
	return graph.NewWithStore(
		ResourceHasher,
		graph_addons.NewMemoryStore[ResourceId, *Resource](),
		options...,
	)
}

func NewGraph(options ...func(*graph.Traits)) Graph {
	return NewGraphWithOptions(append(options,
		graph.Directed(),
	)...,
	)
}

func NewAcyclicGraph(options ...func(*graph.Traits)) Graph {
	return NewGraphWithOptions(append(options, graph.Directed(), graph.PreventCycles())...)
}

func ResourceHasher(r *Resource) ResourceId {
	return r.ID
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
	topo, err := TopologicalSort(g)
	if err != nil {
		return err
	}
	adjacent, err := g.AdjacencyMap()
	if err != nil {
		return err
	}

	var errs []error
	write := func(format string, args ...any) {
		_, err := fmt.Fprintf(w, format, args...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, id := range topo {
		write("%q", id)

		targets := make([]ResourceId, 0, len(adjacent[id]))
		for t := range adjacent[id] {
			targets = append(targets, t)
		}
		sort.Sort(SortedIds(targets))
		if len(targets) > 1 {
			write("\n")
		} else if len(targets) == 1 {
			write(" ")
		}

		for _, t := range targets {
			e := adjacent[id][t]
			// Adjacent edges always have `id` as the source, so just write the target.
			write("-> %q", t)

			if e.Properties.Weight > 1 {
				write(" (weight=%d)", e.Properties.Weight)
			}
			write("\n")
		}
	}
	return nil
}

type IdResolutionError map[ResourceId]error

func (e IdResolutionError) Error() string {
	if len(e) == 1 {
		for id, err := range e {
			return fmt.Sprintf("failed to resolve ID %s: %v", id, err)
		}
	}
	var b strings.Builder
	b.WriteString("failed to resolve IDs:\n")
	for id, err := range e {
		fmt.Fprintf(&b, "  %s: %v\n", id, err)
	}
	return b.String()
}

func ResolveIds(g Graph, ids []ResourceId) ([]*Resource, error) {
	errs := make(IdResolutionError)
	var resources []*Resource
	for _, id := range ids {
		res, err := g.Vertex(id)
		if err != nil {
			errs[id] = err
			continue
		}
		resources = append(resources, res)
	}
	if len(errs) > 0 {
		return resources, errs
	}
	return resources, nil
}

func ResourceEdgeToKeyEdge(re ResourceEdge) Edge {
	return Edge{
		Source:     re.Source.ID,
		Target:     re.Target.ID,
		Properties: re.Properties,
	}
}
