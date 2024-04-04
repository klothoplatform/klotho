package operational_eval

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/klothoplatform/klotho/pkg/engine/path_selection"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/set"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_pathExpandVertex_Key(t *testing.T) {
	assert := assert.New(t)
	v := &pathExpandVertex{
		SatisfactionEdge: construct.SimpleEdge{
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
				SatisfactionEdge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
				Satisfication: knowledgebase.EdgePathSatisfaction{
					Classification: "network",
				},
			},
			mocks: func(mr *MockexpansionRunner, me *MockEdgeExpander, v *pathExpandVertex) {
				input := path_selection.ExpansionInput{
					SatisfactionEdge: construct.ResourceEdge{
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

func Test_pathExpandVertex_getDepsForScripts(t *testing.T) {
	tests := []struct {
		name    string
		v       *pathExpandVertex
		mocks   func(dcap *MockdependencyCapturer)
		wantErr bool
	}{
		{
			name: "add deps from source script",
			v: &pathExpandVertex{
				SatisfactionEdge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
				Satisfication: knowledgebase.EdgePathSatisfaction{
					Source: knowledgebase.PathSatisfactionRoute{
						Script: "myscript",
					},
				},
			},
			mocks: func(dcap *MockdependencyCapturer) {
				dcap.EXPECT().ExecuteDecode("myscript",
					knowledgebase.DynamicValueData{Resource: construct.ResourceId{Name: "s"}},
					&construct.ResourceList{}).Return(nil).Times(1)
			},
		},
		{
			name: "add deps from target script",
			v: &pathExpandVertex{
				SatisfactionEdge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
				Satisfication: knowledgebase.EdgePathSatisfaction{
					Target: knowledgebase.PathSatisfactionRoute{
						Script: "myscript",
					},
				},
			},
			mocks: func(dcap *MockdependencyCapturer) {
				dcap.EXPECT().ExecuteDecode("myscript",
					knowledgebase.DynamicValueData{Resource: construct.ResourceId{Name: "t"}},
					&construct.ResourceList{}).Return(nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			dcap := NewMockdependencyCapturer(ctrl)
			tt.mocks(dcap)
			err := getDepsForScripts(tt.v, dcap)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
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
		mocks   func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty, dcap *MockdependencyCapturer) error
		want    graphChanges
		wantErr bool
	}{
		{
			name: "no temp graph",
			v: &pathExpandVertex{
				SatisfactionEdge: construct.SimpleEdge{
					Source: construct.ResourceId{Name: "s"},
					Target: construct.ResourceId{Name: "t"},
				},
				Satisfication: knowledgebase.EdgePathSatisfaction{
					Classification: "network",
				},
			},
			mocks: func(mockSol *enginetesting.MockSolution, mockKB *MockTemplateKB, mockProperty *MockProperty, dcap *MockdependencyCapturer) error {
				dcap.EXPECT().GetChanges().Times(1)
				return nil
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
			dcap := NewMockdependencyCapturer(ctrl)
			eval := &Evaluator{Solution: mockSol}
			if tt.mocks != nil {
				err := tt.mocks(mockSol, mockKB, mockProperty, dcap)
				if !assert.NoError(err) {
					return
				}
			}
			err := tt.v.Dependencies(eval, dcap)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			mockSol.AssertExpectations(t)
			ctrl.Finish()
		})
	}
}
