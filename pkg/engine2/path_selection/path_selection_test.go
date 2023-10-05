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

func Test_SelectPath(t *testing.T) {
	tests := []struct {
		name     string
		dep      graph.Edge[*construct.Resource]
		edgeData EdgeData
		kbMocks  []mock.Call
		expected []graph.Edge[*construct.Resource]
	}{
		{
			name: "Select path",
			dep: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			kbMocks: []mock.Call{
				{
					Method: "HasDirectPath",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{true},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{
					Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
					Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
				},
			},
		},
		{
			name: "Select path with constraints",
			dep: graph.Edge[*construct.Resource]{
				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.Resource{{ID: construct.ResourceId{Type: "middle"}}},
				},
			},
			kbMocks: []mock.Call{
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "middle"}},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Type: "middle"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "HasDirectPath",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{true},
				},
				{
					Method: "AllPaths",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{
						[][]*knowledgebase.ResourceTemplate{
							{
								createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
								createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle"}),
								createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
							},
							{
								createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
								createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
							},
						},
						nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: []interface{}{
						construct.ResourceId{Type: "source"},
						construct.ResourceId{Type: "middle"}},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: []interface{}{
						construct.ResourceId{Type: "middle"},
						construct.ResourceId{Type: "target"}},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			expected: []graph.Edge[*construct.Resource]{
				{
					Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
					Target: &construct.Resource{ID: construct.ResourceId{Type: "middle"}},
				},
				{
					Source: &construct.Resource{ID: construct.ResourceId{Type: "middle"}},
					Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
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
			paths, err := ctx.SelectPath(test.dep, test.edgeData)
			if !assert.NoError(err, "Expected no error") {
				return
			}
			assert.Equal(test.expected, paths)
			kb.AssertExpectations(t)
		})
	}
}

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

			for _, m := range test.kbMocks {
				kb.On(m.Method, m.Arguments...).Return(m.ReturnArguments...).Once()
				kb.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{})
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

func Test_ctxcontainsUnneccessaryHopsInPath(t *testing.T) {
	tests := []struct {
		name      string
		dep       graph.Edge[*construct.Resource]
		path      []construct.ResourceId
		templates map[string]*knowledgebase.ResourceTemplate
		edgeData  EdgeData
		expected  bool
	}{
		{
			name: "Path does not contain unnecessary hops",
			dep: graph.Edge[*construct.Resource]{

				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "target"},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				":source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				":middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle"}),
				":target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: false,
		},
		{
			name: "Path contains unnecessary hops",
			dep: graph.Edge[*construct.Resource]{

				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "target"},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				":source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				":middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
				":target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: true,
		},
		{
			name: "Path contains unnecessary hops due to constraint",
			dep: graph.Edge[*construct.Resource]{

				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "target"},
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.Resource{{ID: construct.ResourceId{Type: "middle"}}},
				},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				":source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				":middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
				":target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: false,
		},
		{
			name: "Path contains unnecessary hops due duplicate compute in middle",
			dep: graph.Edge[*construct.Resource]{

				Source: &construct.Resource{ID: construct.ResourceId{Type: "source"}},
				Target: &construct.Resource{ID: construct.ResourceId{Type: "target"}},
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle", Namespace: "one"},
				{Type: "middle", Namespace: "two"},
				{Type: "target"},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				":source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				":middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
				":target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := PathSelectionContext{}
			assert.Equal(test.expected, ctx.containsUnneccessaryHopsInPath(test.dep, test.path, test.edgeData, test.templates))
		})
	}
}

func Test_findOptimalPath(t *testing.T) {
	tests := []struct {
		name     string
		paths    []Path
		expected Path
	}{
		{
			name: "Find optimal path",
			paths: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "target"}},
					Weight: 2,
				},
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
					Weight: 2,
				},
			},
			expected: Path{
				Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "target"}},
				Weight: 2,
			},
		},
		{
			name: "Find optimal path with different weights",
			paths: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "target"}},
					Weight: 2,
				},
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
					Weight: 1,
				},
			},

			expected: Path{
				Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
				Weight: 1,
			},
		},
		{

			name: "Find optimal path with same weights and lengths",
			paths: []Path{
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
					Weight: 1,
				},
				{
					Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle2"}, {Type: "target"}},
					Weight: 1,
				},
			},
			expected: Path{
				Nodes:  []construct.ResourceId{{Type: "source"}, {Type: "middle"}, {Type: "target"}},
				Weight: 1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			ctx := PathSelectionContext{}
			assert.Equal(test.expected, ctx.findOptimalPath(test.paths))
		})
	}
}
