package graph

import (
	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"github.com/klothoplatform/klotho/pkg/k2/model"
)

type (
	Graph = graph.Graph[model.URN, model.URN]
	Edge  = graph.Edge[model.URN]
)

func NewGraphWithOptions(options ...func(*graph.Traits)) Graph {
	return graph.NewWithStore(
		UrnHasher,
		graph_addons.NewMemoryStore[model.URN, model.URN](),
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

func UrnHasher(r model.URN) model.URN {
	return r
}

func ResolveDeploymentGroups(g graph.Graph[model.URN, model.URN]) ([][]model.URN, error) {
	sorted, err := graph_addons.TopologicalSort(g, func(a, b model.URN) bool {
		return a.Compare(b) < 0
	})
	if err != nil {
		return nil, err
	}
	var groups [][]model.URN
	var currentGroup []model.URN
	visited := make(map[model.URN]bool)

	for _, node := range sorted {
		if !hasEdges(node, currentGroup, g) {
			currentGroup = append(currentGroup, node)
			visited[node] = true
		} else {
			groups = append(groups, currentGroup)
			currentGroup = []model.URN{node}
		}
	}
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}
	return groups, nil
}

// hasDependencies checks if a node has dependencies in the current group
func hasEdges(node model.URN, group []model.URN, g graph.Graph[model.URN, model.URN]) bool {
	for _, n := range group {
		if _, err := g.Edge(node, n); err == nil {
			return true
		}
		if _, err := g.Edge(n, node); err == nil {
			return true
		}
	}
	return false
}
