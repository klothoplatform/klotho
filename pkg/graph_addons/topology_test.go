package graph_addons

import (
	"slices"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_topologicalSort(t *testing.T) {
	type Edge = graph.Edge[int]
	less := func(a, b int) bool {
		return a < b
	}

	tests := map[string]struct {
		vertices      []int
		edges         []Edge
		expectedOrder []int
		shouldFail    bool
	}{
		"graph with 5 vertices": {
			vertices: []int{1, 2, 3, 4, 5},
			edges: []Edge{
				{Source: 1, Target: 2},
				{Source: 1, Target: 3},
				{Source: 2, Target: 3},
				{Source: 2, Target: 4},
				{Source: 2, Target: 5},
				{Source: 3, Target: 4},
				{Source: 4, Target: 5},
			},
			expectedOrder: []int{1, 2, 3, 4, 5},
		},
		"graph with many possible topological orders": {
			vertices: []int{1, 2, 3, 4, 5, 6, 10, 20, 30, 40, 50, 60},
			edges: []Edge{
				{Source: 1, Target: 10},
				{Source: 2, Target: 20},
				{Source: 3, Target: 30},
				{Source: 4, Target: 40},
				{Source: 5, Target: 50},
				{Source: 6, Target: 60},
			},
			expectedOrder: []int{1, 2, 3, 4, 5, 6, 10, 20, 30, 40, 50, 60},
		},
		"graph with cycle": {
			vertices: []int{1, 2, 3, 4},
			edges: []Edge{ // 1 -> 3 -> 2 -> 4 ↺
				{Source: 1, Target: 3},
				{Source: 3, Target: 2},
				{Source: 2, Target: 4},
				{Source: 4, Target: 1},
			},
			expectedOrder: []int{1, 3, 2, 4},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require, assert := require.New(t), assert.New(t)

			g := graph.New(graph.IntHash, graph.Directed())

			for _, vertex := range test.vertices {
				_ = g.AddVertex(vertex)
			}

			for _, edge := range test.edges {
				require.NoError(
					g.AddEdge(edge.Source, edge.Target, func(ep *graph.EdgeProperties) { *ep = edge.Properties }),
				)
			}

			order, err := TopologicalSort(g, less)

			if test.shouldFail {
				require.Error(err)
				return
			}
			require.NoError(err)

			assert.Equal(test.expectedOrder, order, "regular order doesn't match")

			reverse, err := ReverseTopologicalSort(g, ReverseLess(less))
			require.NoError(err)

			slices.Reverse(test.expectedOrder)
			assert.Equal(test.expectedOrder, reverse, "reverse order doesn't match")
		})
	}
}
