package engine

import (
	"strings"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestFileFormat(t *testing.T) {
	makeGraph := func(elements ...any) construct.Graph {
		return graphtest.MakeGraph(t, construct.NewGraph(), elements...)
	}
	tests := []struct {
		name string
		file FileFormat
		yml  string
	}{
		{
			name: "simple input",
			file: FileFormat{
				Graph: makeGraph(
					"p:t:a -> p:t:b",
				),
				Constraints: constraints.Constraints{
					Application: []constraints.ApplicationConstraint{
						{
							Operator: constraints.AddConstraintOperator,
							Node:     construct.ResourceId{Provider: "p", Type: "t", Name: "a"},
						},
					},
				},
			},
			yml: `constraints:
    - scope: application
      operator: add
      node: p:t:a
resources:
    p:t:a:
    p:t:b:
edges:
    p:t:a -> p:t:b:
outputs: {}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			b, err := yaml.Marshal(tt.file)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(
				strings.TrimSpace(tt.yml),
				strings.TrimSpace(string(b)),
			)

			var got FileFormat
			err = yaml.Unmarshal(b, &got)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.file.Constraints, got.Constraints)
			graphtest.AssertGraphEqual(t, tt.file.Graph, got.Graph, "")
		})
	}
}
