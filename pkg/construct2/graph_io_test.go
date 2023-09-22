package construct2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphToYAML(t *testing.T) {
	makeGraph := func(elements ...any) Graph {
		return MakeGraph(t, NewGraph(), elements...)
	}
	tests := []struct {
		name string
		g    Graph
		yml  string
	}{
		{
			name: "empty graph",
			g:    NewGraph(),
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

			w := &bytes.Buffer{}
			err := GraphToYAML(tt.g, w)
			if !assert.NoError(err) {
				return
			}

			assert.Equal(
				strings.TrimSpace(tt.yml),
				strings.TrimSpace(w.String()),
				"YAML diff",
			)

			g := NewGraph()
			err = AddFromYAML(g, bytes.NewReader(w.Bytes()))
			if !assert.NoError(err) {
				return
			}

			AssertGraphEqual(t, tt.g, g)
		})
	}
}
