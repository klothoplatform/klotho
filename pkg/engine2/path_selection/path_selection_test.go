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

func Test_PathSelection(t *testing.T) {
	tests := []struct {
		name     string
		dep      construct.ResourceEdge
		edgeData EdgeData
		kb       []mock.Call
		want     []construct.ResourceId
		wantErr  bool
	}{
		{
			name: "can select a direct path",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Name: "test2"}),
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "HasDirectPath",
					Arguments: mock.Arguments{
						construct.ResourceId{
							Name: "test",
						},
						construct.ResourceId{
							Name: "test2",
						},
					},
					ReturnArguments: mock.Arguments{
						true,
					},
				},
			},
			want: []construct.ResourceId{
				{Name: "test"},
				{Name: "test2"},
			},
		},
		{
			name: "can select a path with constraints",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test2"}),
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.ResourceId{
						{Provider: "mock", Type: "test", Name: "test3"},
					},
				},
			},
			kb: []mock.Call{
				{
					Method: "HasDirectPath",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{true},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
						},
					},
				},
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "test"},
						construct.ResourceId{Provider: "mock", Type: "test"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil), nil,
					},
				},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "test", Name: "test"},
				{Provider: "mock", Type: "test", Name: "test3"},
				{Provider: "mock", Type: "test", Name: "test2"},
			},
		},
		{
			name: "prefers glue over functional resources",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test2"}),
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "HasDirectPath",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{false},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test2"}, []string{"compute"}),
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test3"}, nil),
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test4"}, nil),
						},
					},
				},
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test2"}, []string{"compute"}),
							},
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test2"}, []string{"compute"}),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
							},
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test3"}, nil),
							},
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test3"}, nil),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test4"}, nil),
							},
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test4"}, nil),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil), nil,
					},
				},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "test", Name: "test"},
				{Provider: "mock", Type: "test3"},
				{Provider: "mock", Type: "test4"},
				{Provider: "mock", Type: "test", Name: "test2"},
			},
		},
		{
			name: "unnecessary hop shortest path gets rejected",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test2"}),
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "HasDirectPath",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{false},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test2"}, []string{"compute"}),
							createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test3"}, []string{"compute"}),
						},
					},
				},
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test2"}, []string{"compute"}),
							},
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test2"}, []string{"compute"}),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test3"}, []string{"compute"}),
							},
							{
								Source: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test3"}, []string{"compute"}),
								Target: createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, nil),
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
				{
					Method: "GetResourceTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						createResourceTemplate(construct.ResourceId{Provider: "mock", Type: "test"}, []string{"compute"}), nil,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			mockKB := enginetesting.MockKB{}
			for _, call := range tt.kb {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			ms := &enginetesting.MockSolution{}
			ms.On("KnowledgeBase").Return(&mockKB)
			got, err := SelectPath(ms, tt.dep, tt.edgeData)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, got)
		})
	}
}

func Test_addResourcesToTempGraph(t *testing.T) {
	tests := []struct {
		name     string
		dep      construct.ResourceEdge
		edgeData EdgeData
		kb       []mock.Call
		want     []construct.ResourceId
	}{
		{
			name: "can add source and destination to graph",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Name: "test2"}),
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{},
					},
				},
			},
			want: []construct.ResourceId{
				{Name: "test"},
				{Name: "test2"},
			},
		},
		{
			name: "can add nodes that must exist to graph",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Name: "test2"}),
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.ResourceId{
						{
							Name: "test3",
						},
					},
				},
			},
			kb: []mock.Call{
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{},
					},
				},
			},
			want: []construct.ResourceId{
				{Name: "test"},
				{Name: "test2"},
				{Name: "test3"},
			},
		},
		{
			name: "can add allowable kb nodes to graph",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Name: "test2"}),
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{
							{
								QualifiedTypeName: "mock:templateResource",
							},
						},
					},
				},
			},
			want: []construct.ResourceId{
				{Name: "test"},
				{Name: "test2"},
				{Provider: "mock", Type: "templateResource"},
			},
		},
		{
			name: "rejects must not exist resources",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Name: "test2"}),
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustNotExist: []construct.ResourceId{
						{
							Provider: "mock",
							Type:     "test3",
						},
					},
				},
			},
			kb: []mock.Call{
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{
							{
								QualifiedTypeName: "mock:test3",
							},
						},
					},
				},
			},
			want: []construct.ResourceId{
				{Name: "test"},
				{Name: "test2"},
			},
		},
		{
			name: "rejects resources which do not satisfy attributes",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Name: "test2"}),
			},
			edgeData: EdgeData{
				Attributes: map[string]any{
					"test":  "test",
					"test2": "test2",
				},
			},
			kb: []mock.Call{
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*knowledgebase.ResourceTemplate{
							{
								QualifiedTypeName: "mock:test3",
								Classification: knowledgebase.Classification{
									Is: []string{
										"test",
									},
								},
							},
							{
								QualifiedTypeName: "mock:test4",
								Classification: knowledgebase.Classification{
									Is: []string{
										"test",
										"test2",
									},
								},
							},
							{
								QualifiedTypeName: "mock:test5",
							},
						},
					},
				},
			},
			want: []construct.ResourceId{
				{Name: "test"},
				{Name: "test2"},
				{Provider: "mock", Type: "test4"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			mockKB := enginetesting.MockKB{}
			for _, call := range tt.kb {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			g := graph.New(
				func(r construct.ResourceId) construct.ResourceId {
					return r
				},
				graph.Directed(),
				graph.Weighted(),
			)

			err := addResourcesToTempGraph(g, tt.dep, tt.edgeData, &mockKB)
			if err != nil {
				t.Errorf("addResourcesToTempGraph() error = %v", err)
				return
			}
			for _, want := range tt.want {
				res, err := g.Vertex(want)
				assert.NoError(err)
				assert.Equal(want, res)
			}
			mockKB.AssertExpectations(t)
		})
	}
}

func Test_addEdgesToTempGraph(t *testing.T) {
	tests := []struct {
		name         string
		dep          construct.ResourceEdge
		initialState []construct.ResourceId
		edgeData     EdgeData
		kb           []mock.Call
		want         []graph.Edge[construct.ResourceId]
	}{
		{
			name: "can add edges to graph",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test2", Name: "test2"}),
			},
			initialState: []construct.ResourceId{
				{Provider: "mock", Type: "test", Name: "test"},
				{Provider: "mock", Type: "test2", Name: "test2"},
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test",
								},
								Target: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test2",
								},
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "test"},
						construct.ResourceId{Provider: "mock", Type: "test2"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			want: []graph.Edge[construct.ResourceId]{
				{
					Source:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test"},
					Target:     construct.ResourceId{Provider: "mock", Type: "test2", Name: "test2"},
					Properties: graph.EdgeProperties{Weight: 0, Attributes: map[string]string{}},
				},
			},
		},
		{
			name: "adds weight for functionality",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test2", Name: "test2"}),
			},
			initialState: []construct.ResourceId{
				{Provider: "mock", Type: "test", Name: "test"},
				{Provider: "mock", Type: "test2", Name: "test2"},
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test",
									Classification:    knowledgebase.Classification{Is: []string{"compute"}},
								},
								Target: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test2",
								},
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "test"},
						construct.ResourceId{Provider: "mock", Type: "test2"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			want: []graph.Edge[construct.ResourceId]{
				{
					Source:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test"},
					Target:     construct.ResourceId{Provider: "mock", Type: "test2", Name: "test2"},
					Properties: graph.EdgeProperties{Weight: 1, Attributes: map[string]string{}},
				},
			},
		},
		{
			name: "adds weight for direct edge only",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test2", Name: "test2"}),
			},
			initialState: []construct.ResourceId{
				{Provider: "mock", Type: "test", Name: "test"},
				{Provider: "mock", Type: "test2", Name: "test2"},
			},
			edgeData: EdgeData{},
			kb: []mock.Call{
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test",
								},
								Target: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test2",
								},
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						construct.ResourceId{Provider: "mock", Type: "test"},
						construct.ResourceId{Provider: "mock", Type: "test2"},
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{
							DirectEdgeOnly: true,
						},
					},
				},
			},
			want: []graph.Edge[construct.ResourceId]{
				{
					Source:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test"},
					Target:     construct.ResourceId{Provider: "mock", Type: "test2", Name: "test2"},
					Properties: graph.EdgeProperties{Weight: 100, Attributes: map[string]string{}},
				},
			},
		},
		{
			name: "adds negative weight for constraints",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test"}),
				Target: construct.CreateResource(construct.ResourceId{Provider: "mock", Type: "test", Name: "test2"}),
			},
			initialState: []construct.ResourceId{
				{Provider: "mock", Type: "test", Name: "test"},
				{Provider: "mock", Type: "test", Name: "test2"},
				{Provider: "mock", Type: "test", Name: "test3"},
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.ResourceId{
						{Provider: "mock", Type: "test", Name: "test3"},
					},
				},
			},
			kb: []mock.Call{
				{
					Method: "Edges",
					ReturnArguments: mock.Arguments{
						[]graph.Edge[*knowledgebase.ResourceTemplate]{
							{
								Source: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test",
								},
								Target: &knowledgebase.ResourceTemplate{
									QualifiedTypeName: "mock:test",
								},
							},
						}, nil,
					},
				},
				{
					Method: "GetEdgeTemplate",
					Arguments: mock.Arguments{
						mock.Anything,
						mock.Anything,
					},
					ReturnArguments: mock.Arguments{
						&knowledgebase.EdgeTemplate{},
					},
				},
			},
			want: []graph.Edge[construct.ResourceId]{
				{
					Source:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test"},
					Target:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test2"},
					Properties: graph.EdgeProperties{Weight: 0, Attributes: map[string]string{}},
				},
				{
					Source:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test3"},
					Target:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test2"},
					Properties: graph.EdgeProperties{Weight: -1000, Attributes: map[string]string{}},
				},
				{
					Source:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test"},
					Target:     construct.ResourceId{Provider: "mock", Type: "test", Name: "test3"},
					Properties: graph.EdgeProperties{Weight: -1000, Attributes: map[string]string{}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			mockKB := enginetesting.MockKB{}
			for _, call := range tt.kb {
				mockKB.On(call.Method, call.Arguments...).Return(call.ReturnArguments...)
			}
			g := graph.New(
				func(r construct.ResourceId) construct.ResourceId {
					return r
				},
				graph.Directed(),
				graph.Weighted(),
			)
			for _, res := range tt.initialState {
				err := g.AddVertex(res)
				if err != nil {
					t.Errorf("addEdgesToTempGraph() error = %v", err)
					return
				}
			}

			err := addEdgesToTempGraph(g, tt.dep, tt.edgeData, &mockKB)
			if err != nil {
				t.Errorf("addEdgesToTempGraph() error = %v", err)
				return
			}
			for _, want := range tt.want {
				res, err := g.Edge(want.Source, want.Target)
				assert.NoError(err)
				assert.Equal(want, res)
			}
			mockKB.AssertExpectations(t)
		})
	}
}

func Test_containsUnneccessaryHopsInPath(t *testing.T) {
	tests := []struct {
		name      string
		dep       construct.ResourceEdge
		path      []construct.ResourceId
		templates map[string]*knowledgebase.ResourceTemplate
		edgeData  EdgeData
		expected  bool
	}{
		{
			name: "Path does not contain unnecessary hops",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Type: "source"}),
				Target: construct.CreateResource(construct.ResourceId{Type: "target"}),
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "target"},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				"source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				"middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle"}),
				"target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: false,
		},
		{
			name: "Path contains unnecessary hops",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Type: "source"}),
				Target: construct.CreateResource(construct.ResourceId{Type: "target"}),
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "target"},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				"source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				"middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
				"target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: true,
		},
		{
			name: "Path contains unnecessary hops due to constraint",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Type: "source"}),
				Target: construct.CreateResource(construct.ResourceId{Type: "target"}),
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "target"},
			},
			edgeData: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist: []construct.ResourceId{{Type: "middle"}},
				},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				"source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				"middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
				"target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: false,
		},
		{
			name: "Path contains unnecessary hops due duplicate compute in middle",
			dep: construct.ResourceEdge{
				Source: construct.CreateResource(construct.ResourceId{Type: "source"}),
				Target: construct.CreateResource(construct.ResourceId{Type: "target"}),
			},
			path: []construct.ResourceId{
				{Type: "source"},
				{Type: "middle"},
				{Type: "middle"},
				{Type: "target"},
			},
			templates: map[string]*knowledgebase.ResourceTemplate{
				"source": createResourceTemplate(construct.ResourceId{Type: "source"}, []string{"source", "compute"}),
				"middle": createResourceTemplate(construct.ResourceId{Type: "middle"}, []string{"middle", "compute"}),
				"target": createResourceTemplate(construct.ResourceId{Type: "target"}, []string{"target", "compute"}),
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			mockKB := enginetesting.MockKB{}
			for key, template := range test.templates {
				mockKB.On("GetResourceTemplate", construct.ResourceId{Type: key}).Return(template, nil)
			}
			assert.Equal(test.expected, containsUnneccessaryHopsInPath(test.dep, test.path, test.edgeData, &mockKB))
			mockKB.AssertExpectations(t)
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
