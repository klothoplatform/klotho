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

func WalkUp[K comparable, T any](g graph.Graph[K, T], start K, f WalkGraphFunc[K]) error {
	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}
	return walk(g, start, f, pred)
}

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

	var stack []K
	for d := range deps[start] {
		stack = append(stack, d)
	}
	visited.Add(start)
	if len(stack) == 0 {
		return nil
	}

	var nerr error
	var current K
	for len(stack) > 0 {
		current, stack = stack[0], stack[1:]
		visited.Add(current)

		err := f(current, nerr)
		if errors.Is(err, StopWalk) {
			return nerr
		}
		if errors.Is(err, SkipPath) {
			continue
		}
		nerr = err

		for d := range deps[current] {
			if visited.Contains(d) {
				continue
			}
			stack = append(stack, d)
		}
	}
	return nerr
}
