package operational_eval

import (
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gomock "go.uber.org/mock/gomock"
)

type dynDataMatcher struct {
	data knowledgebase.DynamicValueData
}

func (d dynDataMatcher) Matches(x interface{}) bool {
	dynData, ok := x.(knowledgebase.DynamicValueData)
	if !ok {
		return false
	}

	return dynData.Resource.Matches(d.data.Resource) && dynData.Path.String() == d.data.Path.String() && reflect.DeepEqual(dynData.Edge, d.data.Edge)
}

func (d dynDataMatcher) String() string {
	return fmt.Sprintf("is equal to %v", d.data)
}

func Test_propertyVertex_evaluateResourceOperational(t *testing.T) {
	rule := &knowledgebase.PropertyRule{
		Value: "test",
	}
	type args struct {
		v   *propertyVertex
		res *construct.Resource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "property rule",
			args: args{
				v: &propertyVertex{
					Ref: construct.PropertyRef{
						Property: "test",
						Resource: construct.ResourceId{Name: "test"},
					},
					Template: &properties.StringProperty{
						PropertyDetails: knowledgebase.PropertyDetails{
							OperationalRule: rule,
						},
					},
				},
				res: &construct.Resource{ID: construct.ResourceId{Name: "test"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			opctx := NewMockOpRuleHandler(ctrl)
			opctx.EXPECT().HandlePropertyRule(*rule).Return(nil).Times(1)
			err := tt.args.v.evaluateResourceOperational(tt.args.res, opctx)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			ctrl.Finish()
		})
	}
}

func Test_propertyVertex_evaluateEdgeOperational(t *testing.T) {
	rule := knowledgebase.OperationalRule{
		If: "test",
	}
	type args struct {
		v   *propertyVertex
		res *construct.Resource
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "property rule",
			args: args{
				v: &propertyVertex{
					Ref: construct.PropertyRef{
						Property: "test",
						Resource: construct.ResourceId{Name: "test"},
					},
					EdgeRules: map[construct.SimpleEdge][]knowledgebase.OperationalRule{
						{
							Source: construct.ResourceId{Name: "test"},
							Target: construct.ResourceId{Name: "test2"},
						}: {
							rule,
						},
					},
				},
				res: &construct.Resource{ID: construct.ResourceId{Name: "test"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			opctx := NewMockOpRuleHandler(ctrl)
			opctx.EXPECT().SetData(knowledgebase.DynamicValueData{
				Resource: tt.args.v.Ref.Resource,
				Edge:     &graph.Edge[construct.ResourceId]{Source: construct.ResourceId{Name: "test"}, Target: construct.ResourceId{Name: "test2"}},
			}).Times(1)
			opctx.EXPECT().HandleOperationalRule(rule).Return(nil).Times(1)
			err := tt.args.v.evaluateEdgeOperational(tt.args.res, opctx)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			ctrl.Finish()
		})
	}
}

func Test_propertyVertex_Dependencies(t *testing.T) {

	tests := []struct {
		name    string
		v       *propertyVertex
		mocks   func(dcap *MockdependencyCapturer, resource *construct.Resource, path construct.PropertyPath)
		wantErr bool
	}{
		{
			name: "property vertex with template",
			v: &propertyVertex{
				Ref: construct.PropertyRef{
					Property: "test",
					Resource: construct.ResourceId{Name: "test"},
				},
				Template: &properties.StringProperty{
					PropertyDetails: knowledgebase.PropertyDetails{
						OperationalRule: &knowledgebase.PropertyRule{
							If: "test",
						},
					},
				},
			},
			mocks: func(dcap *MockdependencyCapturer, resource *construct.Resource, path construct.PropertyPath) {
				dcap.EXPECT().ExecutePropertyRule(dynDataMatcher{
					data: knowledgebase.DynamicValueData{
						Resource: resource.ID,
						Path:     path,
					},
				},
					knowledgebase.PropertyRule{
						If: "test",
					},
				).Return(nil)
			},
		},
		{
			name: "property vertex with edge rules",
			v: &propertyVertex{
				Ref: construct.PropertyRef{
					Property: "test",
					Resource: construct.ResourceId{Name: "test"},
				},
				EdgeRules: map[construct.SimpleEdge][]knowledgebase.OperationalRule{
					{
						Source: construct.ResourceId{Name: "test"},
						Target: construct.ResourceId{Name: "test2"},
					}: {
						{
							If: "testE",
						},
					},
				},
			},
			mocks: func(dcap *MockdependencyCapturer, resource *construct.Resource, path construct.PropertyPath) {
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Resource: resource.ID,
					Edge:     &graph.Edge[construct.ResourceId]{Source: construct.ResourceId{Name: "test"}, Target: construct.ResourceId{Name: "test2"}},
				}, knowledgebase.OperationalRule{
					If: "testE",
				}).Return(nil)
			},
		},

			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					ctrl := gomock.NewController(t)
				dcap := NewMockdependencyCapturer(ctrl)
				resource := &construct.Resource{ID: construct.ResourceId{Name: "test"}, Properties: construct.Properties{
					"test": "test",
				}}
				path, err := resource.PropertyPath("test")
				if !assert.NoError(t, err) {
					return
				}
				tt.mocks(dcap, resource, path)
				testSol := enginetesting.NewTestSolution()
				testSol.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
				err = testSol.RawView().AddVertex(resource)
				if !assert.NoError(t, err) {
					return
				}
				eval := &Evaluator{
					Solution: testSol,
				}
				err = tt.v.Dependencies(eval, dcap)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				ctrl.Finish()
			})
		}
	}
	
=======
func Test_propertyVertex_evaluateConstraints(t *testing.T) {
	id := construct.ResourceId{Provider: "test", Type: "test", Name: "test"}
	type fields struct {
		Ref       construct.PropertyRef
		EdgeRules map[construct.SimpleEdge][]knowledgebase.OperationalRule
	}
	tests := []struct {
		name    string
		fields  fields
		res     *construct.Resource
		mocks   func(mockSol *enginetesting.MockSolution, mockProperty *MockProperty)
		wantErr bool
	}{
		{
			name: "existing value and no constraints",
			fields: fields{
				Ref: construct.PropertyRef{
					Resource: id,
					Property: "test",
				},
			},
			res: &construct.Resource{ID: id, Properties: construct.Properties{"test": "test"}},
			mocks: func(mockSol *enginetesting.MockSolution, mockProperty *MockProperty) {
				mockSol.On("Constraints").Return(&constraints.Constraints{})
				mockSol.On("DataflowGraph").Return(construct.NewGraph())
			},
		},
		{
			name: "no value and no constraints sets default",
			fields: fields{
				Ref: construct.PropertyRef{
					Resource: id,
					Property: "test",
				},
			},
			res: &construct.Resource{ID: id},
			mocks: func(mockSol *enginetesting.MockSolution, mockProperty *MockProperty) {
				mockSol.On("Constraints").Return(&constraints.Constraints{})
				mockSol.On("DataflowGraph").Return(construct.NewGraph())
				mockProperty.EXPECT().GetDefaultValue(gomock.Any(),
					knowledgebase.DynamicValueData{Resource: id}).Return("test", nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSol := &enginetesting.MockSolution{}
			mockProperty := NewMockProperty(ctrl)
			tt.mocks(mockSol, mockProperty)
			v := &propertyVertex{
				Ref:       tt.fields.Ref,
				Template:  mockProperty,
				EdgeRules: tt.fields.EdgeRules,
			}
			err := v.evaluateConstraints(mockSol, tt.res)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			mockSol.AssertExpectations(t)
			ctrl.Finish()
		})
	}
}
