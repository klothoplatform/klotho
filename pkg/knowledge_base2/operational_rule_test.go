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
			want: "02c61dd3cd718a1cb28439705f2291018dbffe8aaa110f4e5eb72b67f0963b4f",
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
			want: "ec11f23efba8646d56563e96ddf8d7963d09cede7290b6242efe49bb68e91c40",
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
