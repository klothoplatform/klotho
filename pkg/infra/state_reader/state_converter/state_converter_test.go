package stateconverter

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func Test_convertKeysToCamelCase(t *testing.T) {
	tests := []struct {
		name string
		data construct.Properties
		want construct.Properties
	}{
		{
			name: "converts keys to camel case",
			data: construct.Properties{
				"urn":  "urn",
				"type": "type",
				"outputs": map[string]interface{}{
					"output_key": "output_value",
				},
			},
			want: construct.Properties{
				"Urn":  "urn",
				"Type": "type",
				"Outputs": map[string]interface{}{
					"OutputKey": "output_value",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got := convertKeysToCamelCase(tt.data)
			assert.Equal(tt.want, got)
		})
	}
}
