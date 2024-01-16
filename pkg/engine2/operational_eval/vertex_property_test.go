package operational_eval

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

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
	ctrl := gomock.NewController(t)
	dcap := NewMockdependencyCapturer(ctrl)

	type fields struct {
		Ref           construct.PropertyRef
		Template      knowledgebase.Property
		EdgeRules     map[construct.SimpleEdge][]knowledgebase.OperationalRule
		ResourceRules map[string][]knowledgebase.OperationalRule
	}
	tests := []struct {
		name    string
		fields  fields
		mocks   func()
		wantErr bool
	}{
		{
			name: "property vertex with template",
			fields: fields{
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
			mocks: func() {
				dcap.EXPECT().ExecutePropertyRule(knowledgebase.DynamicValueData{
					Resource: construct.ResourceId{Name: "test"},
				}, knowledgebase.PropertyRule{
					If: "test",
				}).Return(nil)
			},
		},
		{
			name: "property vertex with edge rules and resource rules",
			fields: fields{
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
				ResourceRules: map[string][]knowledgebase.OperationalRule{
					"test": {
						{
							If: "testR",
						},
					},
				},
			},
			mocks: func() {
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Resource: construct.ResourceId{Name: "test"},
				}, knowledgebase.OperationalRule{
					If: "testR",
				}).Return(nil)
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Resource: construct.ResourceId{Name: "test"},
					Edge:     &graph.Edge[construct.ResourceId]{Source: construct.ResourceId{Name: "test"}, Target: construct.ResourceId{Name: "test2"}},
				}, knowledgebase.OperationalRule{
					If: "testE",
				}).Return(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prop := &propertyVertex{
				Ref:           tt.fields.Ref,
				Template:      tt.fields.Template,
				EdgeRules:     tt.fields.EdgeRules,
				ResourceRules: tt.fields.ResourceRules,
			}
			tt.mocks()
			eval := &Evaluator{}
			err := prop.Dependencies(eval, dcap)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			ctrl.Finish()
		})
	}
}
