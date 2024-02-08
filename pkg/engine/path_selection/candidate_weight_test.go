package path_selection

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/kbtesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_determineCandidateWeight(t *testing.T) {
	// NOTE(gg): this test is a little brittle, since the weights don't really have any meaning other than
	// their relative values. They're made up to get the desired results in path selection.
	tests := []struct {
		name        string
		graph       []any
		resultGraph []any
		src, target string
		id          string
		wantWeight  int
		wantErr     bool
	}{
		{
			name:       "no relation",
			graph:      []any{"p:compute:a -> p:glue:b -> p:glue:c -> p:compute:d"},
			src:        "p:compute:a",
			target:     "p:compute:d",
			id:         "p:compute:e",
			wantWeight: 2,
		},
		{
			name:        "no relation, in result graph",
			graph:       []any{"p:compute:a -> p:glue:b -> p:glue:c -> p:compute:d"},
			resultGraph: []any{"p:compute:e"},
			src:         "p:compute:a",
			target:      "p:compute:d",
			id:          "p:compute:e",
			wantWeight:  11,
		},
		{
			name:       "downstream direct / upstream indirect",
			graph:      []any{"p:compute:a -> p:glue:b -> p:glue:c -> p:compute:d"},
			src:        "p:compute:a",
			target:     "p:compute:d",
			id:         "p:glue:b",
			wantWeight: 21,
		},
		{
			name:       "downstream indirect",
			graph:      []any{"p:compute:a -> p:glue:b -> p:glue:c -> p:glue:d -> p:compute:e"},
			src:        "p:compute:a",
			target:     "p:compute:e",
			id:         "p:glue:c",
			wantWeight: 15,
		},
		{
			name:       "upstream direct",
			graph:      []any{"p:compute:a -> p:glue:b -> p:glue:c -> p:compute:d"},
			src:        "p:compute:a",
			target:     "p:compute:d",
			id:         "p:glue:c",
			wantWeight: 20,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := enginetesting.NewTestSolution()
			ctx.KB.
				On("GetResourceTemplate", mock.MatchedBy(construct.ResourceId{Type: "compute"}.Matches)).
				Return(&knowledgebase.ResourceTemplate{
					Classification: knowledgebase.Classification{
						Is: []string{"compute"},
					},
				}, nil)
			ctx.KB.
				On("GetResourceTemplate", mock.MatchedBy(construct.ResourceId{Type: "glue"}.Matches)).
				Return(&knowledgebase.ResourceTemplate{}, nil)
			ctx.KB.
				On("GetEdgeTemplate", mock.Anything, mock.Anything).
				Return(&knowledgebase.EdgeTemplate{})

			ctx.LoadState(t, tt.graph...)

			resultGraph := graphtest.MakeGraph(t, construct.NewGraph(), tt.resultGraph...)

			undirected, err := BuildUndirectedGraph(ctx.DataflowGraph(), ctx.KnowledgeBase())
			require.NoError(t, err)

			src := graphtest.ParseId(t, tt.src)
			target := graphtest.ParseId(t, tt.target)
			id := graphtest.ParseId(t, tt.id)

			gotWeight, err := determineCandidateWeight(ctx, src, target, id, resultGraph, undirected)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantWeight, gotWeight)
		})
	}
}

func TestBuildUndirectedGraph(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	kb := &kbtesting.MockKB{}
	kb.Mock.
		On("GetResourceTemplate", mock.MatchedBy(construct.ResourceId{Type: "compute"}.Matches)).
		Return(&knowledgebase.ResourceTemplate{
			Classification: knowledgebase.Classification{
				Is: []string{"compute"},
			},
		}, nil)
	kb.Mock.
		On("GetResourceTemplate", mock.MatchedBy(construct.ResourceId{Type: "glue"}.Matches)).
		Return(&knowledgebase.ResourceTemplate{}, nil)

	graph := graphtest.MakeGraph(t, construct.NewGraph(),
		"p:compute:a -> p:glue:b -> p:glue:c -> p:compute:d",
		"p:compute:e",
	)

	got, err := BuildUndirectedGraph(graph, kb)
	require.NoError(err)

	assert.False(got.Traits().IsDirected, "graph should be undirected")
	assert.True(got.Traits().IsWeighted, "graph should be weighted")

	for _, f := range []func(construct.Graph) (int, error){construct.Graph.Order, construct.Graph.Size} {
		want, err := f(graph)
		require.NoError(err)
		got, err := f(got)
		require.NoError(err)
		assert.Equal(want, got)
	}

	getNodes := func(g construct.Graph) []construct.ResourceId {
		adj, err := g.AdjacencyMap()
		require.NoError(err)
		nodes := make([]construct.ResourceId, 0, len(adj))
		for n := range adj {
			nodes = append(nodes, n)
		}
		return nodes
	}
	assert.ElementsMatch(getNodes(graph), getNodes(got))

	wantWeights := map[string]int{
		"p:compute:a -> p:glue:b": 1000, // compute -> unknown
		"p:glue:b -> p:glue:c":    1,    // unknown -> unknown
		"p:glue:c -> p:compute:d": 1000, // unknown -> compute
	}
	for e, w := range wantWeights {
		wantEdge := graphtest.ParseEdge(t, e)
		gotEdge, err := got.Edge(wantEdge.Source, wantEdge.Target)

		if assert.NoError(err) {
			assert.Equal(w, gotEdge.Properties.Weight)
		}
	}
}
