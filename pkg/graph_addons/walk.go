package graph_addons

import (
	"errors"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/set"
)

type WalkGraphFunc[K comparable] func(k K, nerr error) error

var (
	StopWalk = errors.New("stop walk")
	SkipPath = errors.New("skip path")
)

// WalkUp walks up through the graph starting at `start` in BFS order.
func WalkUp[K comparable, T any](g graph.Graph[K, T], start K, f WalkGraphFunc[K]) error {
	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}
	return walk(g, start, f, pred)
}

// WalkDown walks down through the graph starting at `start` in BFS order.
func WalkDown[K comparable, T any](g graph.Graph[K, T], start K, f WalkGraphFunc[K]) error {
	adj, err := g.AdjacencyMap()
	if err != nil {
		return err
	}
	return walk(g, start, f, adj)
}

func walk[K comparable, T any](
	g graph.Graph[K, T],
	start K,
	f WalkGraphFunc[K],
	deps map[K]map[K]graph.Edge[K],
) error {
	visited := make(set.Set[K])
	var queue []K

	for d := range deps[start] {
		queue = append(queue, d)
	}
	visited.Add(start)

	var err error
	var current K
	for len(queue) > 0 {
		current, queue = queue[0], queue[1:]
		visited.Add(current)

		nerr := f(current, err)
		if errors.Is(nerr, StopWalk) {
			return err
		}
		if errors.Is(nerr, SkipPath) {
			continue
		}
		err = nerr

		for d := range deps[current] {
			if visited.Contains(d) {
				continue
			}
			queue = append(queue, d)
		}
	}
	return err
}
