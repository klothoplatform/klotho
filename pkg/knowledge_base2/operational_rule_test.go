package knowledgebase2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func Test_ResourceSelector_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    ResourceSelector
	}{
		{
			name:    "valid resource selector",
			content: `selector: "mock:resource1"`,
			want: ResourceSelector{
				Selector: "mock:resource1",
			},
		},
		{
			name:    "string resource selector",
			content: `mock:resource1`,
			want: ResourceSelector{
				Selector: "mock:resource1",
			},
		},
		{
			name: "resource selector with properties",
			content: `selector: "mock:resource1"
properties:
  name: "test"`,
			want: ResourceSelector{
				Selector: "mock:resource1",
				Properties: map[string]any{
					"name": "test",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			var selector ResourceSelector
			node := &yaml.Node{}

			err := yaml.Unmarshal([]byte(tt.content), node)
			assert.NoError(err, "Expected no error")

			err = selector.UnmarshalYAML(node)
			assert.NoError(err, "Expected no error")
			assert.Equal(tt.want, selector)

		})
	}

}
