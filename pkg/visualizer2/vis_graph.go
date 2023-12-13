package visualizer

import (
	"errors"

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
