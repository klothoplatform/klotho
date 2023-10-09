package construct2

import (
	"sort"

	"github.com/klothoplatform/klotho/pkg/set"
)

// AllDownstreamDependencies returns all downstream dependencies of the given resource.
// Downstream means that for A -> B -> C -> D the downstream dependencies of B are [C, D].
func AllDownstreamDependencies(g Graph, r ResourceId) ([]ResourceId, error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	return allDependencies(adj, r), nil
}

// DirectDownstreamDependencies returns the direct downstream dependencies of the given resource.
// Direct means that for A -> B -> C -> D the direct downstream dependencies of B are [C].
func DirectDownstreamDependencies(g Graph, r ResourceId) ([]ResourceId, error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	var ids []ResourceId
	for d := range adj[r] {
		ids = append(ids, d)
	}
	sort.Sort(sortedIds(ids))

	return ids, nil
}

// AllUpstreamDependencies returns all upstream dependencies of the given resource.
// Upstream means that for A -> B -> C -> D the upstream dependencies of C are [B, A] (in that order).
func AllUpstreamDependencies(g Graph, r ResourceId) ([]ResourceId, error) {
	adj, err := g.PredecessorMap()
	if err != nil {
		return nil, err
	}

	return allDependencies(adj, r), nil
}

// DirectUpstreamDependencies returns the direct upstream dependencies of the given resource.
// Direct means that for A -> B -> C -> D the direct upstream dependencies of C are [B].
func DirectUpstreamDependencies(g Graph, r ResourceId) ([]ResourceId, error) {
	adj, err := g.PredecessorMap()
	if err != nil {
		return nil, err
	}

	var ids []ResourceId
	for d := range adj[r] {
		ids = append(ids, d)
	}
	sort.Sort(sortedIds(ids))

	return ids, nil
}

func allDependencies(deps map[ResourceId]map[ResourceId]Edge, r ResourceId) []ResourceId {
	visited := make(map[ResourceId]struct{})

	var stack []ResourceId
	for d := range deps[r] {
		stack = append(stack, d)
	}
	sort.Sort(sortedIds(stack))

	var ids []ResourceId
	for len(stack) > 0 {
		id := stack[0]
		stack = stack[1:]

		visited[id] = struct{}{}
		ids = append(ids, id)

		var next []ResourceId
		for d := range deps[id] {
			if _, ok := visited[d]; ok {
				continue
			}
			next = append(next, d)
		}
		sort.Sort(sortedIds(next))
		stack = append(stack, next...)
	}

	return ids
}

func Neighbors(g Graph, r ResourceId) (upstream, downstream set.Set[ResourceId], err error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, nil, err
	}

	pred, err := g.PredecessorMap()
	if err != nil {
		return nil, nil, err
	}

	downstream = make(set.Set[ResourceId])
	for d := range adj[r] {
		downstream.Add(d)
	}

	upstream = make(set.Set[ResourceId])
	for u := range pred[r] {
		upstream.Add(u)
	}

	return upstream, downstream, nil
}
