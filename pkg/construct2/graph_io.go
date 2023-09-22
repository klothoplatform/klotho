package construct2

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type ioEdge struct {
	Source ResourceId
	Target ResourceId
}

func (e ioEdge) String() string {
	return fmt.Sprintf("%s -> %s", e.Source, e.Target)
}

func (e ioEdge) MarshalText() (string, error) {
	return e.String(), nil
}

func (e *ioEdge) UnmarshalText(data []byte) error {
	s := string(data)

	source, target, found := strings.Cut(s, " -> ")
	if !found {
		target, source, found = strings.Cut(s, " <- ")
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
	topo, err := ToplogicalSort(g)
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
	for _, source := range topo {
		targets := make([]ResourceId, 0, len(adj[source]))
		for t := range adj[source] {
			targets = append(targets, t)
		}
		sort.Sort(sortedIds(targets))
		for _, target := range targets {
			writeln("  %s -> %s:", source, target)
		}
	}

	return
}

func AddFromYAML(g Graph, r io.Reader) error {
	type graph struct {
		Resouces map[ResourceId]Properties `yaml:"resources"`
		Edges    map[ioEdge]struct{}       `yaml:"edges"`
	}
	var y graph
	if err := yaml.NewDecoder(r).Decode(&y); err != nil {
		return err
	}

	var errs error
	for rid, props := range y.Resouces {
		err := g.AddVertex(&Resource{
			ID:         rid,
			Properties: props,
		})
		errs = errors.Join(errs, err)
	}
	for e := range y.Edges {
		err := g.AddEdge(e.Source, e.Target)
		errs = errors.Join(errs, err)
	}
	return errs
}
