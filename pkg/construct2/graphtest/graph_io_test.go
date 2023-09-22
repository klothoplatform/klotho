package graphtest

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGraphToYAML(t *testing.T) {
	makeGraph := func(elements ...any) construct2.Graph {
		return MakeGraph(t, construct2.NewGraph(), elements...)
	}
	tests := []struct {
		name string
		g    construct2.Graph
		yml  string
	}{
		{
			name: "empty graph",
			g:    construct2.NewGraph(),
			yml: `resources:
edges:`,
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
    p:t:a -> p:t:b:`,
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
    p:t:c -> p:t:a:`,
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
    p:t:c -> p:t:b:`,
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
    p:t:d -> p:t:c:`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			b, err := yaml.Marshal(construct2.YamlGraph{Graph: tt.g})
			if !assert.NoError(err) {
				return
			}

			assert.Equal(
				strings.TrimSpace(tt.yml),
				strings.TrimSpace(string(b)),
				"YAML diff",
			)

			got := construct2.YamlGraph{Graph: construct2.NewGraph()}
			err = yaml.Unmarshal(b, &got)
			if !assert.NoError(err) {
				return
			}

			AssertGraphEqual(t, tt.g, got.Graph)
		})
	}
}
