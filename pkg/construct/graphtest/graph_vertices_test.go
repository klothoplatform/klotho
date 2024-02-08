package graphtest

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToplogicalSort(t *testing.T) {
	makeGraph := func(args ...any) construct.Graph {
		return MakeGraph(t, construct.NewGraph(), args...)
	}
	tests := []struct {
		name  string
		graph construct.Graph
		want  []string
	}{
		{
			name: "simple ordered",
			graph: makeGraph(
				"P:a -> P:b",
				"P:b -> P:c",
			),
			want: []string{
				"P:a",
				"P:b",
				"P:c",
			},
		},
		{
			name: "simple id ordered",
			graph: makeGraph(
				"P:a",
				"P:b",
				"P:c",
			),
			want: []string{
				"P:a",
				"P:b",
				"P:c",
			},
		},
		{
			name: "contains cycle",
			graph: makeGraph(
				"P:a -> P:b",
				"P:b -> P:c",
				"P:c -> P:a",
			),
			want: []string{
				// Chooses the first by ID
				"P:a",
				"P:b",
				"P:c",
			},
		},
		{
			name: "fixed bugged graph non-determinism",
			graph: makeGraph(
				"aws:lambda_function:function -> aws:iam_role:function-ExecutionRole",
				"aws:lambda_function:function -> aws:ecr_image:function-image",
				"aws:lambda_function:function -> aws:log_group:function-log-group",
				"aws:ecr_image:function-image -> aws:ecr_repo:ecr_repo-0",
			),
			want: []string{
				"aws:lambda_function:function", // first topologically
				// middle three sorted by ID alphabetically
				"aws:ecr_image:function-image",
				"aws:iam_role:function-ExecutionRole",
				"aws:log_group:function-log-group",
				"aws:ecr_repo:ecr_repo-0", // last topologically
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert, require := assert.New(t), require.New(t)

			// Repeat to make sure it's fully deterministic
			for i := 0; i < 100; i++ {
				got, err := construct.TopologicalSort(tt.graph)
				require.NoError(err)
				gotStr := make([]string, len(got))
				for i, v := range got {
					gotStr[i] = v.String()
				}
				assert.Equal(tt.want, gotStr, "on iteration %d", i)
			}
		})
	}
}
