package construct2

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dominikbraun/graph"
	"gopkg.in/yaml.v3"
)

type ioEdge struct {
	Source ResourceId
	Target ResourceId
}

func (e ioEdge) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%s -> %s", e.Source, e.Target), nil
}

func (e *ioEdge) UnmarshalYAML(n *yaml.Node) error {
	var s string
	if err := n.Decode(&s); err != nil {
		return err
	}

	source, target, found := strings.Cut(s, "->")
	if !found {
		target, source, found = strings.Cut(s, "<-")
		if !found {
			return errors.New("invalid edge format, expected either `source -> target` or `target <- source`")
		}
	}

	srcErr := e.Source.UnmarshalText([]byte(source))
	tgtErr := e.Target.UnmarshalText([]byte(target))
	return errors.Join(srcErr, tgtErr)
}

// GraphToYAML renders the graph `g` as YAML to `w`.
func GraphToYAML(g Graph, w io.Writer) (errs error) {
	// Use a spanning tree to remove cycles
	tree, err := graph.MinimumSpanningTree(g)
	if err != nil {
		return err
	}
	topo, err := ToplogicalSort(tree)
	if err != nil {
		return err
	}

	adj, err := g.AdjacencyMap()
	if err != nil {
		return err
	}

	// Write the yaml explicitly so we can control the order of the keys
	// for resources, their properties, and the edges.

	write := func(s string, args ...interface{}) {
		_, err := fmt.Fprintf(w, s, args...)
		errs = errors.Join(errs, err)
	}
	writeln := func(s string, args ...interface{}) {
		write(s+"\n", args...)
	}

	writeln("resources:")
	for _, rid := range topo {
		writeln("  %s:", rid)

		r, err := g.Vertex(rid)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		// Sort the keys so the output is in a stable, consistent order
		propKeys := make([]string, 0, len(r.Properties))
		for k := range r.Properties {
			propKeys = append(propKeys, k)
		}
		sort.Strings(propKeys)

		for _, k := range propKeys {
			v := r.Properties[k]
			writeln("    %s: %v", k, v)
		}
	}

	writeln("edges:")
	for _, rid := range topo {
		targets := make([]ResourceId, 0, len(adj[rid]))
		for t := range adj[rid] {
			targets = append(targets, t)
		}
		sort.Slice(targets, func(i, j int) bool {
			return isIdLess(targets[i], targets[j])
		})
		for _, e := range adj[rid] {
			writeln("  %s -> %s", e.Source, e.Target)
		}
	}

	return
}

func GraphFromYAML(r io.Reader) (Graph, error) {
	type graph struct {
		resouces map[ResourceId]Properties
		edges    map[ioEdge]struct{}
	}
	var y graph
	if err := yaml.NewDecoder(r).Decode(&y); err != nil {
		return nil, err
	}

	g := NewGraph()
	var errs error
	for rid, props := range y.resouces {
		err := g.AddVertex(&Resource{
			ID:         rid,
			Properties: props,
		})
		errs = errors.Join(errs, err)
	}
	for e := range y.edges {
		err := g.AddEdge(e.Source, e.Target)
		errs = errors.Join(errs, err)
	}
	return g, errs
}
