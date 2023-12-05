package knowledgebase2

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnique_CanAdd(t *testing.T) {
	a := graphtest.ParseId(t, "p:source:a")
	b := graphtest.ParseId(t, "p:target:b")

	tests := []struct {
		name   string
		unique Unique
		graph  []any
		want   bool
	}{
		{
			name:   "not unique",
			unique: Unique{},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:a -> p:target:b",
			},
			want: true,
		},
		{
			name:   "one-to-one (no match)",
			unique: Unique{Source: true, Target: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
			},
			want: true,
		},
		{
			name:   "one-to-one (match target)",
			unique: Unique{Source: true, Target: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:a -> p:target:x",
			},
			want: false,
		},
		{
			name:   "one-to-one (match target)",
			unique: Unique{Source: true, Target: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:x -> p:target:b",
			},
			want: false,
		},
		{
			name:   "one-to-many (no match)",
			unique: Unique{Source: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
			},
			want: true,
		},
		{
			name:   "one-to-many (match target)",
			unique: Unique{Source: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:a -> p:target:x",
			},
			want: true,
		},
		{
			name:   "one-to-many (match source)",
			unique: Unique{Source: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:x -> p:target:b",
			},
			want: false,
		},
		{
			name:   "many-to-one (no match)",
			unique: Unique{Target: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
			},
			want: true,
		},
		{
			name:   "many-to-one (match target)",
			unique: Unique{Target: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:a -> p:target:x",
			},
			want: false,
		},
		{
			name:   "many-to-one (match source)",
			unique: Unique{Target: true},
			graph: []any{
				"p:source:a",
				"p:target:b",
				"p:source:x -> p:target:b",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := graphtest.MakeGraph(t, construct.NewGraph(), tt.graph...)
			edges, err := graph.Edges()
			require.NoError(t, err)
			got := tt.unique.CanAdd(edges, a, b)
			assert.Equal(t, tt.want, got)
		})
	}
}
