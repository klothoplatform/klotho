package construct2

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	Path []ResourceId

	Dependencies struct {
		Resource ResourceId
		Paths    []Path
		All      set.Set[ResourceId]
	}
)

func (p Path) String() string {
	parts := make([]string, len(p))
	for i, id := range p {
		parts[i] = id.String()
	}
	return strings.Join(parts, " -> ")
}

func (p Path) Contains(id ResourceId) bool {
	for _, pathId := range p {
		if pathId == id {
			return true
		}
	}
	return false
}

func (d *Dependencies) Add(p Path) {
	d.Paths = append(d.Paths, p)
	for _, id := range p {
		if id != d.Resource {
			d.All.Add(id)
		}
	}
}

func newDependencies(
	g Graph,
	start ResourceId,
	skipEdge func(Edge) bool,
	deps map[ResourceId]map[ResourceId]Edge,
) (*Dependencies, error) {
	bfRes, err := bellmanFord(g, start, skipEdge)
	if err != nil {
		return nil, err
	}
	d := &Dependencies{Resource: start, All: make(set.Set[ResourceId])}
	for v := range deps {
		path, err := bfRes.ShortestPath(v)
		if errors.Is(err, graph.ErrTargetNotReachable) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("could not get shortest path from %s to %s: %w", start, v, err)
		}
		d.Add(path)
	}
	return d, nil
}

func UpstreamDependencies(g Graph, start ResourceId, skipEdge func(Edge) bool) (*Dependencies, error) {
	pred, err := g.PredecessorMap()
	if err != nil {
		return nil, err
	}
	return newDependencies(g, start, skipEdge, pred)
}

func DownstreamDependencies(g Graph, start ResourceId, skipEdge func(Edge) bool) (*Dependencies, error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	return newDependencies(g, start, skipEdge, adj)
}

type bellmanFordResult struct {
	source ResourceId
	prev   map[ResourceId]ResourceId
}

func bellmanFord(g Graph, source ResourceId, skipEdge func(Edge) bool) (*bellmanFordResult, error) {
	dist := make(map[ResourceId]int)
	prev := make(map[ResourceId]ResourceId)

	adjacencyMap, err := g.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("could not get adjacency map: %w", err)
	}
	for key := range adjacencyMap {
		dist[key] = math.MaxInt32
	}
	dist[source] = 0

	for i := 0; i < len(adjacencyMap)-1; i++ {
		for key, edges := range adjacencyMap {
			for _, edge := range edges {
				if skipEdge(edge) {
					continue
				}
				newDist := dist[key] + edge.Properties.Weight
				if newDist < dist[edge.Target] {
					dist[edge.Target] = newDist
					prev[edge.Target] = key
				}
			}
		}
	}

	for _, edges := range adjacencyMap {
		for _, edge := range edges {
			if skipEdge(edge) {
				continue
			}
			if newDist := dist[edge.Source] + edge.Properties.Weight; newDist < dist[edge.Target] {
				return nil, errors.New("graph contains a negative-weight cycle")
			}
		}
	}

	return &bellmanFordResult{
		source: source,
		prev:   prev,
	}, nil
}

func (b bellmanFordResult) ShortestPath(target ResourceId) (Path, error) {
	var path []ResourceId
	u := target
	for u != b.source {
		if _, ok := b.prev[u]; !ok {
			return nil, graph.ErrTargetNotReachable
		}
		path = append([]ResourceId{u}, path...)
		u = b.prev[u]
	}
	path = append([]ResourceId{b.source}, path...)
	return path, nil
}
