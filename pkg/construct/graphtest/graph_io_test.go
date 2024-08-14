package graphtest

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGraphToYAML(t *testing.T) {
	makeGraph := func(elements ...any) construct.Graph {
		return MakeGraph(t, construct.NewGraph(), elements...)
	}
	tests := []struct {
		name string
		g    construct.Graph
		yml  string
	}{
		{
			name: "empty graph",
			g:    construct.NewGraph(),
			yml: `resources:
edges:
outputs: {}`,
		},
		{
			name: "simple graph",
			g: makeGraph(
				"p:t:a -> p:t:b",
			),
			yml: `resources:
    p:t:a:
    p:t:b:
edges:
    p:t:a -> p:t:b:
outputs: {}`,
		},
		{
			name: "graph with cycle (no roots)",
			g: makeGraph(
				"p:t:a -> p:t:b",
				"p:t:b -> p:t:c",
				"p:t:c -> p:t:a",
			),
			yml: `resources:
    p:t:a:
    p:t:b:
    p:t:c:
edges:
    p:t:a -> p:t:b:
    p:t:b -> p:t:c:
    p:t:c -> p:t:a:
outputs: {}`,
		},
		{
			name: "graph with cycle (with root)",
			g: makeGraph(
				"p:t:a -> p:t:b",
				"p:t:b -> p:t:c",
				"p:t:c -> p:t:b",
			),
			yml: `resources:
    p:t:a:
    p:t:b:
    p:t:c:
edges:
    p:t:a -> p:t:b:
    p:t:b -> p:t:c:
    p:t:c -> p:t:b:
outputs: {}`,
		},
		{
			name: "graph with cycle (predecessor count precedence)",
			g: makeGraph(
				"p:t:a -> p:t:b",
				"p:t:b -> p:t:c",
				"p:t:c -> p:t:b", // b has 1 predecessor upon cycle
				"p:t:c -> p:t:d",
				"p:t:d -> p:t:c", // c has 2 predecessors upon cycle (d and b)
			),
			yml: `resources:
    p:t:a:
    p:t:b:
    p:t:c:
    p:t:d:
edges:
    p:t:a -> p:t:b:
    p:t:b -> p:t:c:
    p:t:c -> p:t:b:
    p:t:c -> p:t:d:
    p:t:d -> p:t:c:
outputs: {}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			b, err := yaml.Marshal(construct.YamlGraph{Graph: tt.g})
			if !assert.NoError(err) {
				return
			}

			assert.Equal(
				strings.TrimSpace(tt.yml),
				strings.TrimSpace(string(b)),
				"YAML diff",
			)

			got := construct.YamlGraph{Graph: construct.NewGraph()}
			err = yaml.Unmarshal(b, &got)
			if !assert.NoError(err) {
				return
			}

			AssertGraphEqual(t, tt.g, got.Graph, "")
		})
	}
}
