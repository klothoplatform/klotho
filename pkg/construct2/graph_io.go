package construct2

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/yaml_util"
	"gopkg.in/yaml.v3"
)

type YamlGraph struct {
	Graph Graph
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
	topo, err := ToplogicalSort(g.Graph)
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
		sort.Sort(sortedIds(targets))
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
		},
	}, nil
}

func (g *YamlGraph) UnmarshalYAML(n *yaml.Node) error {
	type graph struct {
		Resouces map[ResourceId]Properties `yaml:"resources"`
		Edges    map[IoEdge]struct{}       `yaml:"edges"`
	}
	var y graph
	if err := n.Decode(&y); err != nil {
		return err
	}

	if g.Graph == nil {
		g.Graph = NewGraph()
	}

	var errs error
	for rid, props := range y.Resouces {
		err := g.Graph.AddVertex(&Resource{
			ID:         rid,
			Properties: props,
		})
		errs = errors.Join(errs, err)
	}
	for e := range y.Edges {
		err := g.Graph.AddEdge(e.Source, e.Target)
		errs = errors.Join(errs, err)
	}
	return errs
}

type IoEdge struct {
	Source ResourceId
	Target ResourceId
}

func (e IoEdge) String() string {
	return fmt.Sprintf("%s -> %s", e.Source, e.Target)
}

func (e IoEdge) MarshalText() (string, error) {
	return e.String(), nil
}

func (e *IoEdge) UnmarshalText(data []byte) error {
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

func (g YamlGraph) FixStrings() error {
	updateId := func(path PropertyPathItem) error {
		itemVal := path.Get()

		itemStr, ok := itemVal.(string)
		if !ok {
			return nil
		}
		id := ResourceId{}
		err := id.UnmarshalText([]byte(itemStr))
		if err == nil {
			return path.Set(id)
		}
		propertyRef := PropertyRef{}
		err = propertyRef.UnmarshalText([]byte(itemStr))
		if err == nil {
			return path.Set(propertyRef)
		}
		return nil
	}

	return WalkGraph(g.Graph, func(id ResourceId, resource *Resource, nerr error) error {
		err := resource.WalkProperties(func(path PropertyPath, err error) error {
			err = errors.Join(err, updateId(path))

			if kv, ok := path.Last().(PropertyKVItem); ok {
				err = errors.Join(err, updateId(kv.Key()))
			}

			return err
		})
		return errors.Join(nerr, err)
	})
}
