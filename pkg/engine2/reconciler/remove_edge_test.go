package reconciler

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_RemovePath(t *testing.T) {
	tests := []struct {
		name         string
		source       construct.ResourceId
		target       construct.ResourceId
		mockKB       []mock.Call
		initialState []any
		want         []any
	}{
		{
			name:   "remove path simple",
			source: construct.ResourceId{Provider: "p", Type: "t", Name: "s"},
			target: construct.ResourceId{Provider: "p", Type: "t", Name: "t"},
			mockKB: []mock.Call{
				{
					Method:    "GetResourceTemplate",
					Arguments: []any{mock.Anything},
					ReturnArguments: []any{&knowledgebase.ResourceTemplate{
						Classification: knowledgebase.Classification{Is: []string{string(knowledgebase.Compute)}},
					}, nil},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{&knowledgebase.EdgeTemplate{}},
				},
			},
			initialState: []any{"p:t:s", "p:t:t", "p:t:s -> p:t:t"},
			want:         []any{"p:t:s", "p:t:t"},
		},
		{
			name:   "remove shared path",
			source: construct.ResourceId{Provider: "p", Type: "t", Name: "s"},
			target: construct.ResourceId{Provider: "p", Type: "t", Name: "b"},
			mockKB: []mock.Call{
				{
					Method:    "GetResourceTemplate",
					Arguments: []any{mock.Anything},
					ReturnArguments: []any{&knowledgebase.ResourceTemplate{
						Classification: knowledgebase.Classification{Is: []string{string(knowledgebase.Compute)}},
					}, nil},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{&knowledgebase.EdgeTemplate{}},
				},
				{
					Method:    "GetPathSatisfactionsFromEdge",
					Arguments: []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{
						[]knowledgebase.EdgePathSatisfaction{}, nil,
					},
				},
			},
			initialState: []any{"p:t:s", "p:t:t", "p:t:s -> p:t:t", "p:t:a", "p:t:b", "p:t:a -> p:t:t", "p:t:t -> p:t:b"},
			want:         []any{"p:t:s", "p:t:t", "p:t:a", "p:t:b", "p:t:a -> p:t:t", "p:t:t -> p:t:b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSolution := enginetesting.NewTestSolution()
			for _, call := range tt.mockKB {
				testSolution.KB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			testSolution.LoadState(t, tt.initialState...)
			err := RemovePath(tt.source, tt.target, testSolution)
			if !assert.NoError(err) {
				return
			}
			expect := graphtest.MakeGraph(t, construct.NewGraph(), tt.want...)
			graphtest.AssertGraphEqual(t, expect, testSolution.DataflowGraph(), "graph not expected")
		})
	}
}

func Test_findEdgesUsedInOtherPathSelection(t *testing.T) {
	tests := []struct {
		name         string
		source       construct.ResourceId
		target       construct.ResourceId
		mockKB       []mock.Call
		initialState []any
		want         []construct.SimpleEdge
	}{
		{
			name:   "find shared edges",
			source: construct.ResourceId{Provider: "p", Type: "t", Name: "s"},
			target: construct.ResourceId{Provider: "p", Type: "t", Name: "b"},
			mockKB: []mock.Call{
				{
					Method:    "GetResourceTemplate",
					Arguments: []any{mock.Anything},
					ReturnArguments: []any{&knowledgebase.ResourceTemplate{
						Classification: knowledgebase.Classification{Is: []string{string(knowledgebase.Compute)}},
						PathSatisfaction: knowledgebase.PathSatisfaction{
							AsSource: []knowledgebase.PathSatisfactionRoute{
								{Classification: ""},
							},
							AsTarget: []knowledgebase.PathSatisfactionRoute{
								{Classification: ""},
							},
						},
					}, nil},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{&knowledgebase.EdgeTemplate{}},
				},
				{
					Method:    "GetPathSatisfactionsFromEdge",
					Arguments: []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{
						[]knowledgebase.EdgePathSatisfaction{
							{Classification: ""},
						}, nil,
					},
				},
			},
			initialState: []any{"p:t:s", "p:t:t", "p:t:s -> p:t:t", "p:t:a", "p:t:b", "p:t:a -> p:t:t", "p:t:t -> p:t:b"},
			want: []construct.SimpleEdge{
				{Source: construct.ResourceId{Provider: "p", Type: "t", Name: "a"}, Target: construct.ResourceId{Provider: "p", Type: "t", Name: "t"}},
				{Source: construct.ResourceId{Provider: "p", Type: "t", Name: "t"}, Target: construct.ResourceId{Provider: "p", Type: "t", Name: "b"}},
			},
		},
		{
			name:         "path is superset of deletion path",
			source:       construct.ResourceId{Provider: "p", Type: "t", Name: "s"},
			target:       construct.ResourceId{Provider: "p", Type: "t", Name: "b"},
			initialState: []any{"p:t:s", "p:t:t", "p:t:t -> p:t:s", "p:t:a", "p:t:b", "p:t:a -> p:t:t", "p:t:s -> p:t:b", "p:t:l", "p:t:l -> p:t:b"},
			want: []construct.SimpleEdge{
				{Source: construct.ResourceId{Provider: "p", Type: "t", Name: "l"}, Target: construct.ResourceId{Provider: "p", Type: "t", Name: "b"}},
			},
			mockKB: []mock.Call{
				{
					Method:    "GetResourceTemplate",
					Arguments: []any{mock.Anything},
					ReturnArguments: []any{&knowledgebase.ResourceTemplate{
						Classification: knowledgebase.Classification{Is: []string{string(knowledgebase.Compute)}},
						PathSatisfaction: knowledgebase.PathSatisfaction{
							AsSource: []knowledgebase.PathSatisfactionRoute{
								{Classification: ""},
							},
							AsTarget: []knowledgebase.PathSatisfactionRoute{
								{Classification: ""},
							},
						},
					}, nil},
				},
				{
					Method:          "GetEdgeTemplate",
					Arguments:       []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{&knowledgebase.EdgeTemplate{}},
				},
				{
					Method:    "GetPathSatisfactionsFromEdge",
					Arguments: []any{mock.Anything, mock.Anything},
					ReturnArguments: []any{
						[]knowledgebase.EdgePathSatisfaction{
							{Classification: ""},
						}, nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSolution := enginetesting.NewTestSolution()
			for _, call := range tt.mockKB {
				testSolution.KB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			testSolution.LoadState(t, tt.initialState...)
			paths, err := graph.AllPathsBetween(testSolution.DataflowGraph(), tt.source, tt.target)
			if !assert.NoError(err) {
				return
			}
			edges, err := findEdgesUsedInOtherPathSelection(tt.source, tt.target, nodesInPaths(paths), testSolution)
			if !assert.NoError(err) {
				return
			}
			assert.ElementsMatch(tt.want, edges.ToSlice())
		})
	}
}
