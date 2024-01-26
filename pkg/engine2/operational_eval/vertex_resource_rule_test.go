package operational_eval

import (
	"fmt"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_resourceRuleVertex_Dependencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	dcap := NewMockdependencyCapturer(ctrl)

	tests := []struct {
		name     string
		resource construct.ResourceId
		rule     knowledgebase.AdditionalRule
		mocks    func()
		wantErr  bool
	}{
		{
			name:     "resource rule vertex",
			resource: construct.ResourceId{Name: "test"},
			rule: knowledgebase.AdditionalRule{
				If: "test",
			},
			mocks: func() {
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Resource: construct.ResourceId{Name: "test"},
				}, knowledgebase.OperationalRule{
					If: "test",
				}).Return(nil)
			},
		},
		{
			name:     "resource rule vertex",
			resource: construct.ResourceId{Name: "test"},
			rule: knowledgebase.AdditionalRule{
				If: "test",
				Steps: []knowledgebase.OperationalStep{
					{
						Resource: "string",
					},
				},
			},
			mocks: func() {
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Resource: construct.ResourceId{Name: "test"},
				}, knowledgebase.OperationalRule{
					If: "test",
					Steps: []knowledgebase.OperationalStep{
						{
							Resource: "string",
						},
					},
				}).Return(nil)
			},
		},
		{
			name:     "resource rule vertex dependencies throws error",
			resource: construct.ResourceId{Name: "test"},
			rule: knowledgebase.AdditionalRule{
				If: "test",
			},
			mocks: func() {
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Resource: construct.ResourceId{Name: "test"},
				}, knowledgebase.OperationalRule{
					If: "test",
				}).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			v := &resourceRuleVertex{
				Resource: tt.resource,
				Rule:     tt.rule,
			}
			tt.mocks()
			eval := &Evaluator{}
			err := v.Dependencies(eval, dcap)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			dcap.ctrl.Finish()
		})
	}
}

func Test_resourceRuleVertex_UpdateFrom(t *testing.T) {

	tests := []struct {
		name    string
		initial *resourceRuleVertex
		other   Vertex
		want    resourceRuleVertex
	}{
		{
			name: "empty resource rule vertex",
			initial: &resourceRuleVertex{
				Resource: construct.ResourceId{Name: "test"},
			},
			other: &resourceRuleVertex{
				Resource: construct.ResourceId{Name: "test"},
				Rule: knowledgebase.AdditionalRule{
					If: "test",
				},
			},
			want: resourceRuleVertex{
				Resource: construct.ResourceId{Name: "test"},
				Rule: knowledgebase.AdditionalRule{
					If: "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			tt.initial.UpdateFrom(tt.other)
			assert.Equal(tt.want, *tt.initial)
		})
	}
}

func Test_resourceRuleVertex_evaluateResourceRule(t *testing.T) {
	ctrl := gomock.NewController(t)
	opctx := NewMockOpRuleHandler(ctrl)
	tests := []struct {
		name    string
		v       *resourceRuleVertex
		mocks   func()
		wantErr bool
	}{
		{
			name: "resource rule vertex",
			v: &resourceRuleVertex{
				Resource: construct.ResourceId{Name: "test"},
				Rule: knowledgebase.AdditionalRule{
					If: "test",
				},
			},
			mocks: func() {
				opctx.EXPECT().HandleOperationalRule(knowledgebase.OperationalRule{
					If: "test",
				}).Return(nil)
			},
		},
		{
			name: "resource rule vertex throws error",
			v: &resourceRuleVertex{
				Resource: construct.ResourceId{Name: "test"},
				Rule: knowledgebase.AdditionalRule{
					If: "test",
				},
			},
			mocks: func() {
				opctx.EXPECT().HandleOperationalRule(knowledgebase.OperationalRule{
					If: "test",
				}).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eval := &Evaluator{}
			tt.mocks()
			err := tt.v.evaluateResourceRule(opctx, eval)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			opctx.ctrl.Finish()
		})
	}
}
