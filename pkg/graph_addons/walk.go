package graph_addons

import (
	"errors"

	"github.com/dominikbraun/graph"
)

type WalkGraphFunc[K comparable] func(p Path[K], nerr error) error

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
	var queue []Path[K]
	enqueue := func(current Path[K], next K) {
		if current.Contains(next) {
			// Prevent loops
			return
		}
		queue = append(queue, append(current, next))
	}

	startPath := Path[K]{start}
	for d := range deps[start] {
		enqueue(startPath, d)
	}

	var err error
	var current Path[K]
	for len(queue) > 0 {
		current, queue = queue[0], queue[1:]

		nerr := f(current, err)
		if errors.Is(nerr, StopWalk) {
			return err
		}
		if errors.Is(nerr, SkipPath) {
			continue
		}
		err = nerr

		last := current[len(current)-1]
		for d := range deps[last] {
			enqueue(current, d)
		}
	}
	return err
}
