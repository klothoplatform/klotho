package visualizer

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

// ConnectionGraph facilitates whether resources should be connected for visualization. Resources are only considered
// connected if there are paths that satisfy both "network" and "permissions" classifications.
type ConnectionGraph struct {
	Network, Permissions ClassGraph
	Undirected           construct.Graph
	KB                   knowledgebase.TemplateKB
}

func NewConnectionGraph(g construct.Graph, kb knowledgebase.TemplateKB) (ConnectionGraph, error) {
	network, err := connectionGraphBuilder{kb: kb, class: "network"}.makeGraph(g)
	if err != nil {
		return ConnectionGraph{}, err
	}
	permissions, err := connectionGraphBuilder{kb: kb, class: "permissions"}.makeGraph(g)
	if err != nil {
		return ConnectionGraph{}, err
	}
	undirected := construct.NewGraphWithOptions()
	err = undirected.AddVerticesFrom(g)
	if err != nil {
		return ConnectionGraph{}, err
	}
	err = undirected.AddEdgesFrom(g)
	if err != nil {
		return ConnectionGraph{}, err
	}
	return ConnectionGraph{
		Network:     network,
		Permissions: permissions,
		Undirected:  undirected,
		KB:          kb,
	}, nil
}

func (g ConnectionGraph) ForEachTarget(
	source construct.ResourceId,
	f func(target construct.ResourceId, netPath, permPath construct.Path) error,
) error {
	ids, err := construct.TopologicalSort(g.Network)
	if err != nil {
		return err
	}
	networkPather, err := construct.ShortestPaths(g.Network, source, construct.DontSkipEdges)
	if err != nil {
		return fmt.Errorf("network pather: %w", err)
	}
	permissionsPather, err := construct.ShortestPaths(g.Permissions, source, construct.DontSkipEdges)
	if err != nil {
		return fmt.Errorf("permissions pather: %w", err)
	}
	sourceTmpl, err := g.KB.GetResourceTemplate(source)
	if err != nil {
		return fmt.Errorf("source template: %w", err)
	}
	var ferr error
	for _, target := range ids {
		netPath, err := networkPather.ShortestPath(target)
		if errors.Is(err, graph.ErrTargetNotReachable) {
			continue
		} else if err != nil {
			return fmt.Errorf("network path %s -> %s error: %w", source, target, err)
		}
		if len(netPath) > 2 && !g.Network.PathHasClass(netPath[1:len(netPath)-1]) {
			continue
		}
		permPath, err := permissionsPather.ShortestPath(target)
		if errors.Is(err, graph.ErrTargetNotReachable) {
			continue
		} else if err != nil {
			return fmt.Errorf("permissions path %s -> %s error: %w", source, target, err)
		}
		if len(permPath) > 2 && !g.Permissions.PathHasClass(permPath[1:len(permPath)-1]) {
			continue
		}
		targetTmpl, err := g.KB.GetResourceTemplate(target)
		if err != nil {
			return fmt.Errorf("target template: %w", err)
		}
		if len(netPath) > 2 || len(permPath) > 2 {
			if len(sourceTmpl.PathSatisfaction.AsSource) == 0 || len(targetTmpl.PathSatisfaction.AsTarget) == 0 {
				// A path had to have been expanded, but if either the source or target can't then skip it
				continue
			}
		}
		ferr = errors.Join(ferr, f(target, netPath, permPath))
	}
	return ferr
}

type ClassGraph struct {
	construct.Graph
	Class             string
	MatchingResources set.Set[construct.ResourceId]
}

func (g ClassGraph) PathHasClass(path construct.Path) bool {
	for _, id := range path {
		if g.MatchingResources.Contains(id) {
			return true
		}
	}
	return false
}

type connectionGraphBuilder struct {
	kb       knowledgebase.TemplateKB
	class    string
	matching set.Set[construct.ResourceId]

	// numResources is used as the upper bound of path length, used as the "highWeight" in edgeWeight
	// because it needs to be higher than the longest path containing only lowWeight edges.
	numResources int
}

// makeGraph returns a copy of the given graph with edges weighted according to the classifications.
// This results in a graph whose shortest paths by weight are likely to contain one of those classifications
// if they exist.
func (b connectionGraphBuilder) makeGraph(g construct.Graph) (ClassGraph, error) {
	ids, err := construct.TopologicalSort(g)
	if err != nil {
		return ClassGraph{}, err
	}
	b.numResources = len(ids)
	// Set up matching resources set first, it is used in edgeWeight.
	b.matching = make(set.Set[construct.ResourceId])
	var errs error
	for _, id := range ids {
		tmpl, err := b.kb.GetResourceTemplate(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if b.hasClassification(tmpl.Classification.Is) {
			b.matching.Add(id)
		}
	}
	if errs != nil {
		return ClassGraph{}, errs
	}

	weighted := construct.NewGraph(graph.Weighted())
	err = weighted.AddVerticesFrom(g)
	if err != nil {
		return ClassGraph{}, err
	}
	edges, err := g.Edges()
	if err != nil {
		return ClassGraph{}, err
	}
	for _, edge := range edges {
		weight, err := b.edgeWeight(g, edge)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		err = weighted.AddEdge(edge.Source, edge.Target, graph.EdgeWeight(weight))
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("adding edge %s -> %s: %w", edge.Source, edge.Target, err))
		}
	}
	if errs != nil {
		return ClassGraph{}, errs
	}
	return ClassGraph{Graph: weighted, Class: b.class, MatchingResources: b.matching}, errs
}

func (b connectionGraphBuilder) hasClassification(classes []string) bool {
	for _, class := range classes {
		if class == b.class {
			return true
		}
	}
	return false
}

// edgeWeight returns a low weight for edges in which either source or target has one of the arrow classifications
// and a high weight otherwise.
func (b connectionGraphBuilder) edgeWeight(g construct.Graph, edge construct.Edge) (int, error) {
	if b.matching.Contains(edge.Source) || b.matching.Contains(edge.Target) {
		// Smallest weight that isn't 0 because if there is all matching edges, the shortest path should
		// be the one with the fewest edges.
		return 1, nil
	}

	// Use the number of resources as the high weight, because it needs to be higher than the longest path
	// containing only low weight (1) edges.
	return b.numResources, nil
}
