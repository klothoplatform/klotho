package statereader

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_stateReader_LoadGraph(t *testing.T) {
	tests := []struct {
		name    string
		mocks   func(mockKB *enginetesting.MockKB, ctrl *gomock.Controller)
		state   stateconverter.State
		graph   construct.Graph
		want    []*construct.Resource
		wantErr bool
	}{
		{
			name: "ReadState with no input graph",
			mocks: func(mockKB *enginetesting.MockKB, ctrl *gomock.Controller) {
				mockArnProperty := NewMockProperty(ctrl)
				mockIdProperty := NewMockProperty(ctrl)
				mockKB.On("GetResourceTemplate",
					construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_lambda"},
				).Return(&knowledgebase.ResourceTemplate{
					Properties: knowledgebase.Properties{
						"Arn": mockArnProperty,
						"Id":  mockIdProperty,
					},
				}, nil)
				mockArnProperty.EXPECT().Details().Return(&knowledgebase.PropertyDetails{})
				mockIdProperty.EXPECT().Details().Return(&knowledgebase.PropertyDetails{})
				mockArnProperty.EXPECT().Clone().Return(mockArnProperty)
				mockIdProperty.EXPECT().Clone().Return(mockIdProperty)
				mockArnProperty.EXPECT().SetProperty(gomock.Any(), "arn").Return(nil).Times(1)
				mockIdProperty.EXPECT().SetProperty(gomock.Any(), "id").Return(nil).Times(1)
			},
			state: stateconverter.State{
				construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_lambda"}: construct.Properties{
					"Arn": "arn",
					"Id":  "id",
				},
			},
			want: []*construct.Resource{
				{
					ID: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_lambda"},
					Properties: construct.Properties{
						"Arn": "arn",
						"Id":  "id",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			mockKB := &enginetesting.MockKB{}
			tt.mocks(mockKB, ctrl)
			p := stateReader{
				kb:    mockKB,
				graph: construct.NewGraph(),
			}
			err := p.loadGraph(tt.state)
			if !assert.NoError(err) {
				return
			}
			for _, resource := range tt.want {
				resource, err := p.graph.Vertex(resource.ID)
				if !assert.NoError(err) {
					return
				}
				assert.Equal(resource, resource)
			}
			ctrl.Finish()
		})
	}
}
func Test_stateReader_checkValue(t *testing.T) {
	tests := []struct {
		name    string
		mocks   func(mockpc *MockpropertyCorrelation)
		state   stateconverter.State
		graph   construct.Graph
		wantErr bool
	}{
		{
			name: "ReadState with no input graph",
			mocks: func(mockpc *MockpropertyCorrelation) {
				mockpc.EXPECT().setProperty(gomock.Any(), "Arn", "arn").Return(nil).Times(1)
				mockpc.EXPECT().setProperty(gomock.Any(), "Id", "id").Return(nil).Times(1)
			},
			state: stateconverter.State{
				construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_lambda"}: construct.Properties{
					"Arn": "arn",
					"Id":  "id",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ctrl := gomock.NewController(t)
			mockCorrelator := NewMockpropertyCorrelation(ctrl)
			tt.mocks(mockCorrelator)
			p := stateReader{
				graph: construct.NewGraph(),
			}
			for id := range tt.state {
				resource := &construct.Resource{
					ID:         id,
					Properties: make(construct.Properties),
				}
				err := p.graph.AddVertex(resource)
				if !assert.NoError(err) {
					return
				}
			}
			err := p.loadProperties(tt.state, mockCorrelator)
			if !assert.NoError(err) {
				return
			}
			ctrl.Finish()
		})
	}
}
