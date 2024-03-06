package statereader

import (
	"bytes"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_stateReader_ReadState(t *testing.T) {
	tests := []struct {
		name      string
		templates map[string]statetemplate.StateTemplate
		mocks     func(mockConverter *MockStateConverter, mockKB *enginetesting.MockKB)
		state     []byte
		graph     construct.Graph
		want      []*construct.Resource
		wantErr   bool
	}{
		{
			name: "ReadState with no input graph",
			templates: map[string]statetemplate.StateTemplate{
				"aws:lambda/Function:Function": {
					QualifiedTypeName: "aws:lambda_function",
					IaCQualifiedType:  "aws:lambda/Function:Function",
					PropertyMappings: map[string]string{
						"arn": "Arn",
						"id":  "Id",
					},
				},
			},
			mocks: func(mockConverter *MockStateConverter, mockKB *enginetesting.MockKB) {
				bytesReader := bytes.NewReader([]byte(`fake state`))
				mockConverter.EXPECT().ConvertState(bytesReader).Return(stateconverter.State{
					construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_lambda"}: construct.Properties{
						"Arn": "arn",
						"Id":  "id",
					},
				}, nil)
			},
			state: []byte(`fake state`),
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
			mockConverter := NewMockStateConverter(ctrl)
			mockKB := &enginetesting.MockKB{}
			tt.mocks(mockConverter, mockKB)
			p := stateReader{
				templates: tt.templates,
				kb:        mockKB,
				converter: mockConverter,
			}
			got, err := p.ReadState(tt.state, tt.graph)
			if !assert.NoError(err) {
				return
			}
			for _, resource := range tt.want {
				resource, err := got.Vertex(resource.ID)
				if !assert.NoError(err) {
					return
				}
				assert.Equal(resource, resource)
			}
		})
	}
}
