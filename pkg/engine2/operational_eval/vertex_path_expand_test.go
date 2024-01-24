package operational_eval

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_pathExpandVertex_Key(t *testing.T) {
	assert := assert.New(t)
	v := &pathExpandVertex{
		Edge: construct.SimpleEdge{
			Source: construct.ResourceId{Name: "test"},
			Target: construct.ResourceId{Name: "test"},
		},
		Satisfication: knowledgebase.EdgePathSatisfaction{
			Classification: "network",
		},
	}
	assert.Equal(Key{
		Edge: construct.SimpleEdge{
			Source: construct.ResourceId{Name: "test"},
			Target: construct.ResourceId{Name: "test"},
		},
		PathSatisfication: knowledgebase.EdgePathSatisfaction{
			Classification: "network",
		},
	}, v.Key())
}

func Test_pathExpandVertex_runEvaluation(t *testing.T) {
	resultGraph := construct.NewGraph()
	sResource := &construct.Resource{ID: construct.ResourceId{Name: "s"}}
	tResource := &construct.Resource{ID: construct.ResourceId{Name: "t"}}
	err := resultGraph.AddVertex(sResource)
	if err != nil {
		t.Fatal(err)
	}
	if err := resultGraph.AddVertex(tResource); err != nil {
		t.Fatal(err)
	}
	if err := resultGraph.AddEdge(sResource.ID, tResource.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		v       *pathExpandVertex
		mocks   func(mr *MockexpansionRunner, me *MockEdgeExpander, v *pathExpandVertex)
		wantErr bool
	}{
		{
			name: "run evaluation",
			v: &pathExpandVertex{
				Edge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
				Satisfication: knowledgebase.EdgePathSatisfaction{
					Classification: "network",
				},
			},
			mocks: func(mr *MockexpansionRunner, me *MockEdgeExpander, v *pathExpandVertex) {
				input := path_selection.ExpansionInput{
					Dep: construct.ResourceEdge{
						Source: sResource,
						Target: tResource,
					},
					Classification: "network",
				}
				mr.EXPECT().getExpansionsToRun(v).Return([]path_selection.ExpansionInput{
					input,
				}, nil).Times(1)
				result := path_selection.ExpansionResult{
					Edges: []graph.Edge[construct.ResourceId]{
						{Source: construct.ResourceId{Name: "s"}, Target: construct.ResourceId{Name: "t"}},
					},
					Graph: resultGraph,
				}
				me.EXPECT().ExpandEdge(input).Return(result, nil).Times(1)
				mr.EXPECT().addResourcesAndEdges(result, input, v).Times(1)
				mr.EXPECT().addSubExpansion(result, input, v).Times(1)
				mr.EXPECT().consumeExpansionProperties(input).Times(1)
				mr.EXPECT().handleResultProperties(v, result).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockRunner := NewMockexpansionRunner(ctrl)
			mockEdgeExpander := NewMockEdgeExpander(ctrl)
			tt.mocks(mockRunner, mockEdgeExpander, tt.v)
			eval := &Evaluator{}
			err := tt.v.runEvaluation(eval, mockRunner, mockEdgeExpander)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			ctrl.Finish()
		})
	}
}

func Test_pathExpandVertex_addDepsFromProps(t *testing.T) {
	tests := []struct {
		name         string
		v            *pathExpandVertex
		res          construct.ResourceId
		dependencies []construct.ResourceId
		mocks        func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty) error
		want         graphChanges
		wantErr      bool
	}{
		{
			name: "add deps from props",
			v: &pathExpandVertex{
				Edge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
			},
			res: construct.ResourceId{Name: "s"},
			dependencies: []construct.ResourceId{
				{Name: "u"},
			},
			mocks: func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty) error {
				resource := &construct.Resource{ID: construct.ResourceId{Name: "s"}}
				resultGraph := construct.NewGraph()
				err := resultGraph.AddVertex(resource)
				if err != nil {
					return err
				}

				mockSol.On("KnowledgeBase").Return(mockKB).Times(2)
				mockSol.On("DataflowGraph").Return(resultGraph).Once()
				mockKB.EXPECT().GetResourceTemplate(construct.ResourceId{Name: "s"}).Return(
					&knowledgebase.ResourceTemplate{
						Properties: knowledgebase.Properties{
							"test": mockProperty,
						},
					}, nil).Times(1)
				mockProperty.EXPECT().Details().Return(&knowledgebase.PropertyDetails{
					OperationalRule: &knowledgebase.PropertyRule{},
				}).Times(1)

				mockSol.On("RawView").Return(resultGraph).Once()
				mockProperty.EXPECT().Validate(resource, construct.ResourceId{Name: "u"}, gomock.Any()).Return(nil).Times(1)
				return nil
			},
			want: graphChanges{
				nodes: map[Key]Vertex{},
				edges: map[Key]set.Set[Key]{
					{Edge: construct.SimpleEdge{
						Source: construct.ResourceId{Name: "s"},
						Target: construct.ResourceId{Name: "t"},
					}}: set.SetOf(
						Key{Ref: construct.PropertyRef{
							Resource: construct.ResourceId{Name: "s"},
							Property: "test",
						}},
					),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			mockSol := &enginetesting.MockSolution{}
			mockKB := NewMockTemplateKB(ctrl)
			mockProperty := NewMockProperty(ctrl)
			eval := &Evaluator{Solution: mockSol}
			err := tt.mocks(mockSol, mockKB, mockProperty)
			if !assert.NoError(err) {
				return
			}
			changes := newChanges()
			err = tt.v.addDepsFromProps(eval, changes, tt.res, tt.dependencies)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, changes)
			mockSol.AssertExpectations(t)
			ctrl.Finish()
		})
	}
}

func Test_pathExpandVertex_addDepsFromEdge(t *testing.T) {
	tests := []struct {
		name    string
		v       *pathExpandVertex
		edge    construct.Edge
		mocks   func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty) error
		want    graphChanges
		wantErr bool
	}{
		{
			name: "add deps from props",
			v: &pathExpandVertex{
				Edge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
			},
			edge: construct.Edge{
				Source: construct.ResourceId{Name: "f"},
				Target: construct.ResourceId{Name: "l"},
			},
			mocks: func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty) error {
				resource := &construct.Resource{ID: construct.ResourceId{Provider: "f", Type: "f", Name: "f"}}
				resultGraph := construct.NewGraph()
				err := resultGraph.AddVertex(resource)
				if err != nil {
					return err
				}
				mockSol.On("KnowledgeBase").Return(mockKB).Times(2)
				mockSol.On("DataflowGraph").Return(resultGraph).Once()
				mockSol.On("RawView").Return(resultGraph).Once()
				mockKB.EXPECT().GetEdgeTemplate(construct.ResourceId{Name: "f"},
					construct.ResourceId{Name: "l"}).Return(&knowledgebase.EdgeTemplate{
					OperationalRules: []knowledgebase.OperationalRule{
						{
							ConfigurationRules: []knowledgebase.ConfigurationRule{
								{
									Resource: "f:f",
									Config: knowledgebase.Configuration{
										Field: "field1",
									},
								},
							},
						},
					},
				}).Times(1)
				mockKB.EXPECT().GetResourceTemplate(construct.ResourceId{Provider: "f", Type: "f", Name: "f"}).Return(
					&knowledgebase.ResourceTemplate{
						Properties: knowledgebase.Properties{
							"field1": mockProperty,
						},
					}, nil,
				).Times(1)
				return nil
			},
			want: graphChanges{
				nodes: map[Key]Vertex{},
				edges: map[Key]set.Set[Key]{
					{Ref: construct.PropertyRef{
						Resource: construct.ResourceId{Provider: "f", Type: "f", Name: "f"},
						Property: "field1",
					}}: set.SetOf(
						Key{Edge: construct.SimpleEdge{
							Source: construct.ResourceId{Name: "s"},
							Target: construct.ResourceId{Name: "t"},
						}},
					),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			mockSol := &enginetesting.MockSolution{}
			mockKB := NewMockTemplateKB(ctrl)
			mockProperty := NewMockProperty(ctrl)
			eval := &Evaluator{Solution: mockSol}
			err := tt.mocks(mockSol, mockKB, mockProperty)
			if !assert.NoError(err) {
				return
			}
			changes := newChanges()
			err = tt.v.addDepsFromEdge(eval, changes, tt.edge)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, changes)
			mockSol.AssertExpectations(t)
			ctrl.Finish()
		})
	}
}

func Test_getDepsForPropertyRef(t *testing.T) {
	tests := []struct {
		name        string
		res         construct.ResourceId
		propertyRef string
		mocks       func(mockSol *enginetesting.MockSolution) error
		want        set.Set[Key]
	}{
		{
			name:        "get deps for top level property ref",
			res:         construct.ResourceId{Name: "test"},
			propertyRef: "field",
			mocks: func(mockSol *enginetesting.MockSolution) error {
				resource := &construct.Resource{
					ID:         construct.ResourceId{Name: "test"},
					Properties: construct.Properties{},
				}
				resultGraph := construct.NewGraph()
				err := resultGraph.AddVertex(resource)
				if err != nil {
					return err
				}
				mockSol.On("KnowledgeBase").Return(&enginetesting.MockKB{}).Once()
				mockSol.On("DataflowGraph").Return(resultGraph).Once()
				return nil
			},
			want: set.SetOf(
				Key{Ref: construct.PropertyRef{
					Resource: construct.ResourceId{Name: "test"},
					Property: "field",
				}},
			),
		},
		{
			name:        "get deps for top nested property ref that resolves",
			res:         construct.ResourceId{Name: "test"},
			propertyRef: "field#field2",
			mocks: func(mockSol *enginetesting.MockSolution) error {
				resource := &construct.Resource{
					ID: construct.ResourceId{Name: "test"},
					Properties: construct.Properties{
						"field": construct.ResourceId{Name: "test2"},
					},
				}
				resource2 := &construct.Resource{
					ID:         construct.ResourceId{Name: "test2"},
					Properties: construct.Properties{},
				}
				resultGraph := construct.NewGraph()
				err := resultGraph.AddVertex(resource)
				if err != nil {
					return err
				}
				err = resultGraph.AddVertex(resource2)
				if err != nil {
					return err
				}
				mockSol.On("KnowledgeBase").Return(&enginetesting.MockKB{}).Once()
				mockSol.On("DataflowGraph").Return(resultGraph).Once()
				return nil
			},
			want: set.SetOf(
				Key{Ref: construct.PropertyRef{
					Resource: construct.ResourceId{Name: "test"},
					Property: "field",
				}},
				Key{Ref: construct.PropertyRef{
					Resource: construct.ResourceId{Name: "test2"},
					Property: "field2",
				}},
			),
		},
		{
			name:        "get deps for top nested property ref that does not resolve",
			res:         construct.ResourceId{Name: "test"},
			propertyRef: "field#field2",
			mocks: func(mockSol *enginetesting.MockSolution) error {
				resource := &construct.Resource{
					ID:         construct.ResourceId{Name: "test"},
					Properties: construct.Properties{},
				}
				resultGraph := construct.NewGraph()
				err := resultGraph.AddVertex(resource)
				if err != nil {
					return err
				}
				mockSol.On("KnowledgeBase").Return(&enginetesting.MockKB{}).Once()
				mockSol.On("DataflowGraph").Return(resultGraph).Once()
				return nil
			},
			want: set.SetOf(
				Key{Ref: construct.PropertyRef{
					Resource: construct.ResourceId{Name: "test"},
					Property: "field",
				}},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mockSol := &enginetesting.MockSolution{}
			err := tt.mocks(mockSol)
			if !assert.NoError(err) {
				return
			}
			got := getDepsForPropertyRef(mockSol, tt.res, tt.propertyRef)
			assert.Equal(tt.want, got)
			mockSol.AssertExpectations(t)
		})
	}
}

func Test_pathExpandVertex_Dependencies(t *testing.T) {
	tests := []struct {
		name    string
		v       *pathExpandVertex
		mocks   func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty) error
		want    graphChanges
		wantErr bool
	}{
		{
			name: "no temp graph",
			v: &pathExpandVertex{
				Edge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
				Satisfication: knowledgebase.EdgePathSatisfaction{
					Classification: "network",
				},
			},
			want: newChanges(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			mockSol := &enginetesting.MockSolution{}
			mockKB := NewMockTemplateKB(ctrl)
			mockProperty := NewMockProperty(ctrl)
			eval := &Evaluator{Solution: mockSol}
			if tt.mocks != nil {
				err := tt.mocks(mockSol, mockKB, mockProperty)
				if !assert.NoError(err) {
					return
				}
			}
			changes, err := tt.v.Dependencies(eval)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tt.want, changes)
			mockSol.AssertExpectations(t)
			ctrl.Finish()
		})
	}
}
