package construct2

import "sort"

func AllDownstreamDependencies(g Graph, r ResourceId) ([]ResourceId, error) {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	return allDependencies(adj, r), nil
}

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

func AllUpstreamDependencies(g Graph, r ResourceId) ([]ResourceId, error) {
	adj, err := g.PredecessorMap()
	if err != nil {
		return nil, err
	}

	return allDependencies(adj, r), nil
}

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
