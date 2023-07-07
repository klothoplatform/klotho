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

func Test_Engine_Run(t *testing.T) {
	tests := []struct {
		name        string
		constructs  []core.Construct
		edges       []constraints.Edge
		constraints map[constraints.ConstraintScope][]constraints.Constraint
		want        coretesting.ResourcesExpectation
	}{
		{
			name: "sample exec unit -> orm case",
			constructs: []core.Construct{
				&core.ExecutionUnit{Name: "compute"},
				&core.Orm{Name: "orm"},
			},
			edges: []constraints.Edge{
				{
					Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
					Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "orm"},
				},
			},
			constraints: map[constraints.ConstraintScope][]constraints.Constraint{
				constraints.ConstructConstraintScope: {
					&constraints.ConstructConstraint{
						Operator: constraints.EqualsConstraintOperator,
						Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "orm"},
						Type:     "mock3",
					},
					&constraints.ConstructConstraint{
						Operator: constraints.EqualsConstraintOperator,
						Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
						Type:     "mock1",
					},
				},
				constraints.EdgeConstraintScope: {
					&constraints.EdgeConstraint{
						Operator: constraints.MustContainConstraintOperator,
						Target: constraints.Edge{
							Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
							Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "orm"},
						},
						Node: core.ResourceId{Provider: "mock", Type: "mock2", Name: "Corm"},
					},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"mock:mock1:compute",
					"mock:mock2:Corm",
					"mock:mock3:orm",
				},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock1:compute", Destination: "mock:mock2:Corm"},
					{Source: "mock:mock2:Corm", Destination: "mock:mock3:orm"},
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

			cg := core.NewConstructGraph()
			for _, c := range tt.constructs {
				cg.AddConstruct(c)
			}
			for _, e := range tt.edges {
				cg.AddDependency(e.Source, e.Target)
			}

			engine.LoadContext(cg, tt.constraints, "test")
			dag, err := engine.Run()
			tt.want.Assert(t, dag)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
		})
	}
}
