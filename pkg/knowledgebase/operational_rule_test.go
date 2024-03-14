package knowledgebase

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

func TestAdditionalRule_Hash(t *testing.T) {
	tests := []struct {
		name string
		rule AdditionalRule
		want string
	}{
		{
			name: "simple rule",
			rule: AdditionalRule{
				If: "string",
				Steps: []OperationalStep{
					{
						Resource: "string",
					},
				},
			},
			want: "121890af892b7324a133b58b36e4e54a03a192f5268546a56f799cd60ba3fbdb",
		},
		{
			name: "simple rule 2",
			rule: AdditionalRule{
				If: "string",
				Steps: []OperationalStep{
					{
						Resource: "string2",
					},
				},
			},
			want: "a23bc24879d4b78eddef0f4c779fd7b26cc7c8c04cc65cd4b29683342b36961c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			got := tt.rule.Hash()
			assert.Equal(tt.want, got)
		})
	}
}
