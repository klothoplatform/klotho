package operational_eval

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_edgeVertex_Dependencies(t *testing.T) {

	tests := []struct {
		name    string
		v       *edgeVertex
		mocks   func(dcap *MockdependencyCapturer, v *edgeVertex)
		wantErr bool
	}{
		{
			name: "happy path",
			v: &edgeVertex{
				Edge: construct.Edge{
					Source: construct.ResourceId{Name: "source"},
					Target: construct.ResourceId{Name: "target"},
				},
				Rules: []knowledgebase.OperationalRule{
					{
						If: "First",
					},
					{
						If: "Second",
					},
				},
			},
			mocks: func(dcap *MockdependencyCapturer, v *edgeVertex) {
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Edge: &construct.Edge{Source: v.Edge.Source, Target: v.Edge.Target},
				}, v.Rules[0]).Times(1)
				dcap.EXPECT().ExecuteOpRule(knowledgebase.DynamicValueData{
					Edge: &construct.Edge{Source: v.Edge.Source, Target: v.Edge.Target},
				}, v.Rules[1]).Times(1)
				dcap.EXPECT().GetChanges().Times(1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			dcap := NewMockdependencyCapturer(ctrl)
			tt.mocks(dcap, tt.v)
			eval := NewEvaluator(enginetesting.NewTestSolution())
			err := tt.v.Dependencies(eval, dcap)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			ctrl.Finish()
		})
	}
}
