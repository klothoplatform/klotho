package graphtest

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ShortestPaths(t *testing.T) {
	tests := []struct {
		name     string
		graph    []any
		source   string
		skipEdge func(construct.Edge) bool
		wantPath string
		wantErr  bool
	}{
		{
			name: "single path",
			graph: []any{
				"p:t:1 -> p:t:2 -> p:t:3",
			},
			source:   "p:t:1",
			wantPath: "p:t:1 -> p:t:2 -> p:t:3",
		},
		{
			name: "multiple paths",
			graph: []any{
				"p:t:1 -> p:t:2 -> p:t:3",
				"p:t:1 -> p:t:3",
			},
			source:   "p:t:1",
			wantPath: "p:t:1 -> p:t:3",
		},
		{
			name: "has self loop",
			graph: []any{
				"p:t:1 -> p:t:2 -> p:t:3",
				"p:t:1 -> p:t:1",
			},
			source:   "p:t:1",
			wantPath: "p:t:1 -> p:t:2 -> p:t:3",
		},
		{
			name: "has cycle",
			graph: []any{
				"p:t:1 -> p:t:2 -> p:t:3",
				"p:t:3 -> p:t:1",
			},
			source:   "p:t:1",
			wantPath: "p:t:1 -> p:t:2 -> p:t:3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipEdge == nil {
				tt.skipEdge = construct.DontSkipEdges
			}
			g := MakeGraph(t, construct.NewGraph(), tt.graph...)
			r, err := construct.ShortestPaths(g, ParseId(t, tt.source), tt.skipEdge)
			require.NoError(t, err)

			expectPath := ParsePath(t, tt.wantPath)
			got, err := r.ShortestPath(expectPath[len(expectPath)-1])
			if tt.wantErr {
				assert.Error(t, err)
				return
			} else if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, expectPath, got)
		})
	}
}
