package visualizer

import (
	"errors"
	"sort"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	VisResource struct {
		ID construct.ResourceId

		Tag      string
		Parent   construct.ResourceId
		Children set.Set[construct.ResourceId]
	}

	VisEdgeData struct {
		PathResources set.Set[construct.ResourceId]
	}

	VisGraph graph.Graph[construct.ResourceId, *VisResource]
)

func NewVisGraph(options ...func(*graph.Traits)) VisGraph {
	return VisGraph(graph.NewWithStore(
		func(r *VisResource) construct.ResourceId { return r.ID },
		graph_addons.NewMemoryStore[construct.ResourceId, *VisResource](),
		append(options,
			graph.Directed(),
		)...,
	))
}

func ConstructToVis(g construct.Graph) (VisGraph, error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	vis := NewVisGraph()
	var errs error
	for id := range adj {
		errs = errors.Join(errs, vis.AddVertex(&VisResource{ID: id}))
	}
	if errs != nil {
		return nil, errs
	}
	for source, targets := range adj {
		for target := range targets {
			errs = errors.Join(errs, vis.AddEdge(source, target))
		}
	}
	return vis, errs
}

func (d VisEdgeData) MarshalYAML() (interface{}, error) {
	res := d.PathResources.ToSlice()
	sort.Sort(construct.SortedIds(res))
	return map[string]any{
		// TODO infacopilot frontend currently just uses 'path' as the colledction of
		// additional resources to show in the graph for that edge. We have more information
		// we could give, but for compatibility until more is added to the frontend, just flatten
		// everything and call it 'path'.
		"path": res,
	}, nil
}

func VertexAncestors(g VisGraph, id construct.ResourceId) (set.Set[construct.ResourceId], error) {
	ancestors := make(set.Set[construct.ResourceId])
	var err error
	for ancestor := id; !ancestor.IsZero() && err == nil; {
		ancestors.Add(ancestor)

		var ancestorVert *VisResource
		ancestorVert, err = g.Vertex(ancestor)
		if ancestorVert != nil {
			ancestor = ancestorVert.Parent
		}
	}
	return ancestors, err
}
