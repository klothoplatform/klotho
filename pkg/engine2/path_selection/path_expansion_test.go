package path_selection

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ExpandEdge(t *testing.T) {
	tests := []struct {
		name       string
		edge       graph.Edge[*construct.Resource]
		validPath  Path
		expected   []graph.Edge[*construct.Resource]
		edgeData   EdgeData
		mocks      []mock.Call
		graphMocks []mock.Call
	}{
		{
			name: "Expand edge with direct connection",
			edge: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{
					ID:         construct.ResourceId{Name: "source"},
					Properties: construct.Properties{"testKey": "testValue"},
				},
				Target: &construct.Resource{
					ID:         construct.ResourceId{Name: "destination"},
					Properties: construct.Properties{"dest": "dest"},
				},
			},
			mocks: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "source"},
						construct.ResourceId{Name: "destination"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			validPath: Path{
				Nodes: []construct.ResourceId{
					{Name: "source"},
					{Name: "destination"},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{
					Source: &construct.Resource{
						ID:         construct.ResourceId{Name: "source"},
						Properties: construct.Properties{"testKey": "testValue"},
					},
					Target: &construct.Resource{
						ID:         construct.ResourceId{Name: "destination"},
						Properties: construct.Properties{"dest": "dest"},
					},
				},
			},
		},
		{
			name: "Expand edge creates single resource in the middle, no matching constraint",
			edge: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{
					ID:         construct.ResourceId{Name: "source"},
					Properties: construct.Properties{"testKey": "testValue"},
				},
				Target: &construct.Resource{
					ID:         construct.ResourceId{Name: "destination"},
					Properties: construct.Properties{"dest": "dest"},
				},
			},
			validPath: Path{
				Nodes: []construct.ResourceId{
					{Name: "source"},
					{Type: "middle"},
					{Name: "destination"},
				},
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.Resource{
						{ID: construct.ResourceId{Provider: "p", Type: "t", Name: "middle"}},
					},
				},
			},
			mocks: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "source"},
						construct.ResourceId{Type: "middle"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "middle", Name: "middle_source_destination"},
						construct.ResourceId{Name: "destination"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{
					Source: &construct.Resource{
						ID:         construct.ResourceId{Name: "source"},
						Properties: construct.Properties{"testKey": "testValue"},
					},
					Target: &construct.Resource{
						ID: construct.ResourceId{Type: "middle", Name: "middle_source_destination"},
					},
				},
				{
					Source: &construct.Resource{
						ID: construct.ResourceId{Type: "middle", Name: "middle_source_destination"},
					},
					Target: &construct.Resource{
						ID:         construct.ResourceId{Name: "destination"},
						Properties: construct.Properties{"dest": "dest"},
					},
				},
			},
		},
		{
			name: "Expand edge creates constraint node for node must exist",
			edge: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{
					ID:         construct.ResourceId{Name: "source"},
					Properties: construct.Properties{"testKey": "testValue"},
				},
				Target: &construct.Resource{
					ID:         construct.ResourceId{Name: "destination"},
					Properties: construct.Properties{"dest": "dest"},
				},
			},
			validPath: Path{
				Nodes: []construct.ResourceId{
					{Name: "source"},
					{Provider: "p", Type: "t"},
					{Name: "destination"},
				},
			},
			mocks: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "source"},
						construct.ResourceId{Provider: "p", Type: "t"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "p", Type: "t", Name: "middle"},
						construct.ResourceId{Name: "destination"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.Resource{
						{ID: construct.ResourceId{Provider: "p", Type: "t", Name: "middle"}, Properties: construct.Properties{"test": "test"}},
					},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{
					Source: &construct.Resource{
						ID:         construct.ResourceId{Name: "source"},
						Properties: construct.Properties{"testKey": "testValue"},
					},
					Target: &construct.Resource{
						ID: construct.ResourceId{Provider: "p", Type: "t", Name: "middle"}, Properties: construct.Properties{"test": "test"},
					},
				},
				{
					Source: &construct.Resource{
						ID: construct.ResourceId{Provider: "p", Type: "t", Name: "middle"}, Properties: construct.Properties{"test": "test"},
					},
					Target: &construct.Resource{
						ID:         construct.ResourceId{Name: "destination"},
						Properties: construct.Properties{"dest": "dest"},
					},
				},
			},
		},
		{
			name: "Expand edge can reuse upstream",
			edge: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{
					ID: construct.ResourceId{Name: "source"},
				},
				Target: &construct.Resource{
					ID: construct.ResourceId{Name: "destination"},
				},
			},
			validPath: Path{
				Nodes: []construct.ResourceId{
					{Name: "source"},
					{Provider: "p", Type: "t"},
					{Name: "destination"},
				},
			},
			mocks: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "source"},
						construct.ResourceId{Provider: "p", Type: "t"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							Reuse: knowledgebase.ReuseUpstream,
						},
					},
				},
			},
			graphMocks: []mock.Call{
				{
					Method: "Downstream",
					Arguments: mock.Arguments{
						&construct.Resource{
							ID: construct.ResourceId{Name: "source"},
						},
						3,
					},
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							{
								ID:         construct.ResourceId{Provider: "p", Type: "t", Name: "reused"},
								Properties: construct.Properties{"test": "test2"},
							},
						},
						nil,
					},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{

					Source: &construct.Resource{
						ID: construct.ResourceId{Provider: "p", Type: "t", Name: "reused"}, Properties: construct.Properties{"test": "test2"},
					},
					Target: &construct.Resource{
						ID: construct.ResourceId{Name: "destination"},
					},
				},
			},
		},
		{
			name: "Expand edge can reuse downstream",
			edge: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{
					ID: construct.ResourceId{Name: "source"},
				},
				Target: &construct.Resource{
					ID: construct.ResourceId{Name: "destination"},
				},
			},
			validPath: Path{
				Nodes: []construct.ResourceId{
					{Name: "source"},
					{Provider: "p", Type: "t"},
					{Name: "destination"},
				},
			},
			mocks: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Name: "source"},
						construct.ResourceId{Provider: "p", Type: "t"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "p", Type: "t", Name: "t_source_destination"},
						construct.ResourceId{Name: "destination"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							Reuse: knowledgebase.ReuseDownstream,
						},
					},
				},
			},
			graphMocks: []mock.Call{
				{
					Method: "Upstream",
					Arguments: mock.Arguments{
						&construct.Resource{
							ID: construct.ResourceId{Name: "destination"},
						},
						3,
					},
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							{
								ID:         construct.ResourceId{Provider: "p", Type: "t", Name: "reused"},
								Properties: construct.Properties{"test": "test2"},
							},
						},
						nil,
					},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{
					Source: &construct.Resource{
						ID: construct.ResourceId{Name: "source"},
					},
					Target: &construct.Resource{
						ID: construct.ResourceId{Provider: "p", Type: "t", Name: "reused"}, Properties: construct.Properties{"test": "test2"},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			kb := &enginetesting.MockKB{}

			for _, mock := range test.mocks {
				kb.On(mock.Method, mock.Arguments...).Return(mock.ReturnArguments...).Once()
			}

			graph := &enginetesting.MockGraph{}

			for _, mock := range test.graphMocks {
				graph.On(mock.Method, mock.Arguments...).Return(mock.ReturnArguments...).Once()
			}
			ctx := PathSelectionContext{
				KB:    kb,
				Graph: graph,
			}
			result, err := ctx.ExpandEdge(test.edge, test.validPath, test.edgeData)
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != len(test.expected) {
				t.Fatalf("Expected %d edges, got %d", len(test.expected), len(result))
			}
			assert.ElementsMatch(test.expected, result, "Expected expanded edges to equal expected")
			kb.AssertExpectations(t)
			graph.AssertExpectations(t)
		})
	}
}
