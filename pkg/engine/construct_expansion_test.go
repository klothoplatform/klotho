package engine

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func Test_constructExpansion(t *testing.T) {
	tests := []struct {
		name          string
		constraint    constraints.ConstructConstraint
		functionality core.Functionality
		want          []coretesting.ResourcesExpectation
	}{
		{
			name: "simple",
			constraint: constraints.ConstructConstraint{
				Operator:   constraints.EqualsConstraintOperator,
				Target:     core.ResourceId{Name: "compute"},
				Attributes: map[string]any{},
			},
			functionality: core.Compute,
			want: []coretesting.ResourcesExpectation{
				{
					Nodes: []string{
						"mock:mock1:",
					},
					Deps: []coretesting.StringDep{},
				},
				{
					Nodes: []string{
						"mock:mock2:",
					},
					Deps: []coretesting.StringDep{},
				},
			},
		},
		{
			name: "serverless",
			constraint: constraints.ConstructConstraint{
				Operator: constraints.EqualsConstraintOperator,
				Target:   core.ResourceId{Name: "compute"},
				Attributes: map[string]any{
					"serverless": nil,
				},
			},
			functionality: core.Compute,
			want: []coretesting.ResourcesExpectation{
				{
					Nodes: []string{
						"mock:mock1:",
						"mock:mock3:",
					},
					Deps: []coretesting.StringDep{
						{Source: "mock:mock1:", Destination: "mock:mock3:"},
					},
				},
				{
					Nodes: []string{
						"mock:mock2:",
						"mock:mock3:",
					},
					Deps: []coretesting.StringDep{
						{Source: "mock:mock2:", Destination: "mock:mock3:"},
					},
				},
			},
		},
		{
			name: "highly available and serverless",
			constraint: constraints.ConstructConstraint{
				Operator: constraints.EqualsConstraintOperator,
				Target:   core.ResourceId{Name: "compute"},
				Attributes: map[string]any{
					"serverless":       nil,
					"highly_available": nil,
				},
			},
			functionality: core.Compute,
			want: []coretesting.ResourcesExpectation{
				{
					Nodes: []string{
						"mock:mock1:",
						"mock:mock3:",
						"mock:mock4:",
					},
					Deps: []coretesting.StringDep{
						{Source: "mock:mock1:", Destination: "mock:mock3:"},
						{Source: "mock:mock1:", Destination: "mock:mock4:"},
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
			}, enginetesting.MockKB, core.ListAllConstructs())
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument
			graphs, err := engine.expandConstruct(tt.constraint, tt.functionality)
			if !assert.NoError(err) {
				return
			}
			if !assert.Len(graphs, len(tt.want)) {
				return
			}
			for i, g := range graphs {
				tt.want[i].Assert(t, g)
			}
		})
	}
}
