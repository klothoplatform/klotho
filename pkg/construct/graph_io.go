package construct

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/yaml_util"
	"gopkg.in/yaml.v3"
)

type YamlGraph struct {
	Graph   Graph
	Outputs map[string]Output
}

// nullNode is used to render as nothing in the YAML output
// useful for empty mappings, for example instead of `resources: {}`
// it would render as `resources:`. A small change, but helps reduce
// the visual clutter.
var nullNode = &yaml.Node{
	Kind:  yaml.ScalarNode,
	Tag:   "!!null",
	Value: "",
}

func (g YamlGraph) MarshalYAML() (interface{}, error) {
	topo, err := TopologicalSort(g.Graph)
	if err != nil {
		return nil, err
	}

	adj, err := g.Graph.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	var errs error

	resources := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	for _, rid := range topo {
		r, err := g.Graph.Vertex(rid)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if r.Imported {
			r.Properties["imported"] = r.Imported
		}
		props, err := yaml_util.MarshalMap(r.Properties, func(a, b string) bool { return a < b })
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		resources.Content = append(resources.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: rid.String(),
			},
			props,
		)
	}
	if len(resources.Content) == 0 {
		resources = nullNode
	}

	edges := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	for _, source := range topo {
		targets := make([]ResourceId, 0, len(adj[source]))
		for t := range adj[source] {
			targets = append(targets, t)
		}
		sort.Sort(SortedIds(targets))
		for _, target := range targets {
			edges.Content = append(edges.Content,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: fmt.Sprintf("%s -> %s", source, target),
				},
				nullNode)
		}
	}
	if len(edges.Content) == 0 {
		edges = nullNode
	}

	outputs := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	for name, output := range g.Outputs {
		outputs.Content = append(outputs.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: name,
			})

		outputMap := &yaml.Node{
			Kind: yaml.MappingNode,
		}

		if !output.Ref.IsZero() {
			outputMap.Content = append(outputMap.Content,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "ref",
				},
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: output.Ref.String(),
				},
			)
		} else {
			value := &yaml.Node{}
			err = value.Encode(output.Value)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}

			outputMap.Content = append(outputMap.Content,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "value",
				},
				value,
			)
		}
		outputs.Content = append(outputs.Content, outputMap)
	}

	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "resources",
			},
			resources,
			{
				Kind:  yaml.ScalarNode,
				Value: "edges",
			},
			edges,
			{
				Kind:  yaml.ScalarNode,
				Value: "outputs",
			},
			outputs,
		},
	}, nil
}

func (g *YamlGraph) UnmarshalYAML(n *yaml.Node) error {
	type graph struct {
		Resources map[ResourceId]Properties `yaml:"resources"`
		Edges     map[SimpleEdge]struct{}   `yaml:"edges"`
		Outputs   map[string]Output         `yaml:"outputs"`
	}
	var y graph
	if err := n.Decode(&y); err != nil {
		return err
	}

	if g.Graph == nil {
		g.Graph = NewGraph()
	}

	var errs error
	for rid, props := range y.Resources {
		var imported bool
		if imp, ok := props["imported"]; ok {
			val, ok := imp.(bool)
			if !ok {
				errs = errors.Join(errs, fmt.Errorf("unable to parse imported value as boolean for resource %s", rid))
				// Don't continue here so that the vertex is still added, otherwise it could erroneously cause failures in the edge copying
			}
			imported = val
			delete(props, "imported")
		}
		err := g.Graph.AddVertex(&Resource{
			ID:         rid,
			Properties: props,
			Imported:   imported,
		})
		errs = errors.Join(errs, err)
	}
	for e := range y.Edges {
		err := g.Graph.AddEdge(e.Source, e.Target)
		errs = errors.Join(errs, err)
	}

	if g.Outputs == nil {
		g.Outputs = make(map[string]Output)
	}
	for name, output := range y.Outputs {
		g.Outputs[name] = Output{Ref: output.Ref, Value: output.Value}
	}

	return errs
}

type SimpleEdge struct {
	Source ResourceId
	Target ResourceId
}

func (e SimpleEdge) String() string {
	return fmt.Sprintf("%s -> %s", e.Source, e.Target)
}

func (e SimpleEdge) MarshalText() (string, error) {
	return e.String(), nil
}

func (e SimpleEdge) Less(other SimpleEdge) bool {
	if e.Source != other.Source {
		return ResourceIdLess(e.Source, other.Source)
	}
	return ResourceIdLess(e.Target, other.Target)
}

func (e *SimpleEdge) Parse(s string) error {
	source, target, found := strings.Cut(s, " -> ")
	if !found {
		target, source, found = strings.Cut(s, " <- ")
		if !found {
			return errors.New("invalid edge format, expected either `source -> target` or `target <- source`")
		}
	}
	return errors.Join(
		e.Source.Parse(source),
		e.Target.Parse(target),
	)
}

func (e *SimpleEdge) Validate() error {
	return errors.Join(e.Source.Validate(), e.Target.Validate())
}

func (e *SimpleEdge) UnmarshalText(data []byte) error {
	if err := e.Parse(string(data)); err != nil {
		return err
	}
	return e.Validate()
}

func (e SimpleEdge) ToEdge() Edge {
	return Edge{
		Source: e.Source,
		Target: e.Target,
	}
}

func EdgeKeys[V any](m map[SimpleEdge]V) []SimpleEdge {
	keys := make([]SimpleEdge, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Less(keys[j])
	})
	return keys
}
