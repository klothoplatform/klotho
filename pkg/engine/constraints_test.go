package engine

import (
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestApplyConstraints(t *testing.T) {
	tests := []struct {
		name           string
		init           []any
		constraints    constraints.Constraints
		want           enginetesting.ExpectedGraphs
		resourceChecks func(t *testing.T, ctx *enginetesting.TestSolution)
		wantErr        bool
	}{
		{
			name: "add resource",
			constraints: constraints.Constraints{
				Application: []constraints.ApplicationConstraint{
					{Operator: constraints.AddConstraintOperator, Node: graphtest.ParseId(t, "p:t:test")},
				},
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"p:t:test"},
				Deployment: []any{"p:t:test"},
			},
		},
		{
			name: "import resource",
			constraints: constraints.Constraints{
				Application: []constraints.ApplicationConstraint{
					{Operator: constraints.ImportConstraintOperator, Node: graphtest.ParseId(t, "p:t:test")},
				},
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"p:t:test"},
				Deployment: []any{"p:t:test"},
			},
			resourceChecks: func(t *testing.T, ctx *enginetesting.TestSolution) {
				res, err := ctx.RawView().Vertex(graphtest.ParseId(t, "p:t:test"))
				require.NoError(t, err)
				require.True(t, res.Imported)
			},
		},
		{
			name: "add edge",
			init: []any{"p:t:A", "p:t:B"},
			constraints: constraints.Constraints{
				Edges: []constraints.EdgeConstraint{
					{
						Operator: constraints.MustExistConstraintOperator,
						Target: constraints.Edge{
							Source: graphtest.ParseId(t, "p:t:A"),
							Target: graphtest.ParseId(t, "p:t:B"),
						},
					},
				},
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"p:t:A -> p:t:B"},
				Deployment: []any{"p:t:A -> p:t:B"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			ctx := enginetesting.NewTestSolution()
			ctx.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			ctx.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)

			ctx.On("MakeResourcesOperational", mock.Anything).Return(construct.ResourceIdChangeResults(nil), nil)
			ctx.On("MakeEdgeOperational", mock.Anything, mock.Anything).Return(nil, nil, nil)
			ctx.LoadState(t, tt.init...)
			ctx.Constr = tt.constraints

			err := ApplyConstraints(ctx)
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			tt.want.AssertEqual(t, ctx)
		})
	}
}
