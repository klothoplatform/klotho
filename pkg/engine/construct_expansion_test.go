package engine

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func Test_constructExpansion(t *testing.T) {
	tests := []struct {
		name       string
		constraint constraints.ConstructConstraint
		c          construct.Construct
		want       []coretesting.ResourcesExpectation
	}{
		{
			name: "simple",
			constraint: constraints.ConstructConstraint{
				Operator:   constraints.EqualsConstraintOperator,
				Target:     construct.ResourceId{Name: "compute"},
				Attributes: map[string]any{},
			},
			c: &types.ExecutionUnit{Name: "eu_1"},
			want: []coretesting.ResourcesExpectation{
				{
					Nodes: []string{
						"mock:mock1:mock1-eu_1",
					},
					Deps: []coretesting.StringDep{},
				},
				{
					Nodes: []string{
						"mock:mock2:mock2-eu_1",
					},
					Deps: []coretesting.StringDep{},
				},
			},
		},
		{
			name: "serverless",
			constraint: constraints.ConstructConstraint{
				Operator: constraints.EqualsConstraintOperator,
				Target:   construct.ResourceId{Name: "compute"},
				Attributes: map[string]any{
					"serverless": nil,
				},
			},
			c: &types.ExecutionUnit{Name: "eu_1"},
			want: []coretesting.ResourcesExpectation{
				{
					Nodes: []string{
						"mock:mock1:mock1-eu_1",
						"mock:mock3:mock3-eu_1",
					},
					Deps: []coretesting.StringDep{
						{Source: "mock:mock1:mock1-eu_1", Destination: "mock:mock3:mock3-eu_1"},
					},
				},
				{
					Nodes: []string{
						"mock:mock2:mock2-eu_1",
						"mock:mock3:mock3-eu_1",
					},
					Deps: []coretesting.StringDep{
						{Source: "mock:mock2:mock2-eu_1", Destination: "mock:mock3:mock3-eu_1"},
					},
				},
			},
		},
		{
			name: "highly available and serverless",
			constraint: constraints.ConstructConstraint{
				Operator: constraints.EqualsConstraintOperator,
				Target:   construct.ResourceId{Name: "compute"},
				Attributes: map[string]any{
					"serverless":       nil,
					"highly_available": nil,
				},
			},
			c: &types.ExecutionUnit{Name: "eu_1"},
			want: []coretesting.ResourcesExpectation{
				{
					Nodes: []string{
						"mock:mock1:mock1-eu_1",
						"mock:mock3:mock3-eu_1",
						"mock:mock4:mock4-eu_1",
					},
					Deps: []coretesting.StringDep{
						{Source: "mock:mock1:mock1-eu_1", Destination: "mock:mock3:mock3-eu_1"},
						{Source: "mock:mock1:mock1-eu_1", Destination: "mock:mock4:mock4-eu_1"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mp := &enginetesting.MockProvider{}
			engine := NewEngine(map[string]provider.Provider{
				mp.Name(): mp,
			}, enginetesting.MockKB, types.ListAllConstructs())
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument
			solutions, err := engine.expandConstruct(tt.constraint.Type, tt.constraint.Attributes, tt.c)
			if !assert.NoError(err) {
				return
			}
			if !assert.Len(solutions, len(tt.want)) {
				return
			}
			for i, sol := range solutions {
				tt.want[i].Assert(t, sol.Graph)
			}
		})
	}
}
