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

func Test_DetermineCorrectPaths(t *testing.T) {
	tests := []struct {
		name     string
		dep      graph.Edge[*construct.Resource]
		edgedata EdgeData
		kbMocks  []mock.Call
		expected []Path
	}{
		{
			name: "Determine correct paths",
			dep: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			kbMocks: []mock.Call{
				{
					Method: "AllPaths",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{[][]*knowledgebase.ResourceTemplate{
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
					},
						nil,
					},
				},
			},
			expected: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "target"}},
					Weight: 2,
				},
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
					Weight: 2,
				},
			},
		},
		{
			name: "Determine correct paths with constraints",
			dep: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			edgedata: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist:    []construct.Resource{{ID: construct.ResourceId{Type: "middle"}}},
					NodeMustNotExist: []construct.Resource{{ID: construct.ResourceId{Type: "middle2"}}},
				},
			},
			kbMocks: []mock.Call{
				{
					Method: "AllPaths",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{[][]*knowledgebase.ResourceTemplate{
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "middle2"}, []string{"middle2"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
					},
						nil,
					},
				},
			},
			expected: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
					Weight: 2,
				},
			},
		},
		{
			name: "Determine correct paths with attributes",
			dep: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			edgedata: EdgeData{
				Attributes: map[string]any{
					"serverless": true,
				},
			},
			kbMocks: []mock.Call{
				{
					Method: "AllPaths",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{[][]*knowledgebase.ResourceTemplate{
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "middle2"}, []string{"serverless"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
					},
						nil,
					},
				},
			},
			expected: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle2"}, {Type: "target"}},
					Weight: 2,
				},
			},
		},
		{
			name: "Determine correct paths filters unnecessary hops",
			dep: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			kbMocks: []mock.Call{
				{
					Method: "AllPaths",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{[][]*knowledgebase.ResourceTemplate{
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
						{
							createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
							createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
						},
					},
						nil,
					},
				},
			},
			expected: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "target"}},
					Weight: 2,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			kb := &enginetesting.MockKB{}

			for _, mock := range test.kbMocks {
				kb.On(mock.Method, mock.Arguments...).Return(mock.ReturnArguments...).Once()
			}

			ctx := PathSelectionContext{
				KB:    kb,
				Graph: &enginetesting.MockGraph{},
			}
			paths, err := ctx.determineCorrectPaths(test.dep, test.edgedata)
			if !assert.NoError(err, "Expected no error") {
				return
			}
			assert.Equal(test.expected, paths)
		})
	}
}

func createResourceTemplate(id construct.ResourceId, classifications []string) *knowledgebase.ResourceTemplate {
	return &knowledgebase.ResourceTemplate{
		QualifiedTypeName: id.QualifiedTypeName(),
		Classification: knowledgebase.Classification{
			Is: classifications,
		},
	}
}
