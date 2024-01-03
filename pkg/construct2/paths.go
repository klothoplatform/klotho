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

func (p Path) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Path) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), " -> ")
	*p = make(Path, len(parts))
	for i, part := range parts {
		var id ResourceId
		err := id.UnmarshalText([]byte(part))
		if err != nil {
			return err
		}
		(*p)[i] = id
	}
	return nil
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

type ShortestPather interface {
	ShortestPath(target ResourceId) (Path, error)
}

func ShortestPaths(
	g Graph,
	source ResourceId,
	skipEdge func(Edge) bool,
) (ShortestPather, error) {
	return bellmanFord(g, source, skipEdge)
}

func DontSkipEdges(_ Edge) bool {
	return false
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
				if edge.Source == edge.Target {
					continue
				}
				edgeWeight := edge.Properties.Weight
				if !g.Traits().IsWeighted {
					edgeWeight = 1
				}

				newDist := dist[key] + edgeWeight
				if newDist < dist[edge.Target] {
					dist[edge.Target] = newDist
					prev[edge.Target] = key
				} else if newDist == dist[edge.Target] && ResourceIdLess(key, prev[edge.Target]) {
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
			edgeWeight := edge.Properties.Weight
			if !g.Traits().IsWeighted {
				edgeWeight = 1
			}
			if newDist := dist[edge.Source] + edgeWeight; newDist < dist[edge.Target] {
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
		if len(path) > 5000 {
			// This is "slow" but if there's this many path elements, something's wrong
			// and this debug info will be useful.
			for i, e := range path {
				for j := i - 1; j >= 0; j-- {
					if path[j] == e {
						return nil, fmt.Errorf("path contains a cycle: %s", Path(path[j:i+1]))
					}
				}
			}
			return nil, errors.New("path too long")
		}
		path = append([]ResourceId{u}, path...)
		u = b.prev[u]
	}
	path = append([]ResourceId{b.source}, path...)
	return path, nil
}
