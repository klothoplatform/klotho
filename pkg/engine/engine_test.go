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

func Test_Engine_Run(t *testing.T) {
	tests := []struct {
		name        string
		constructs  []construct.Construct
		edges       []constraints.Edge
		constraints map[constraints.ConstraintScope][]constraints.Constraint
		want        coretesting.ResourcesExpectation
	}{
		{
			name: "sample exec unit -> orm case",
			constructs: []construct.Construct{
				&types.ExecutionUnit{Name: "compute"},
				&types.Orm{Name: "orm"},
			},
			edges: []constraints.Edge{
				{
					Source: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
					Target: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "orm"},
				},
			},
			constraints: map[constraints.ConstraintScope][]constraints.Constraint{
				constraints.ConstructConstraintScope: {
					&constraints.ConstructConstraint{
						Operator: constraints.EqualsConstraintOperator,
						Target:   construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "orm"},
						Type:     "mock3",
					},
					&constraints.ConstructConstraint{
						Operator: constraints.EqualsConstraintOperator,
						Target:   construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
						Type:     "mock1",
					},
				},
				constraints.EdgeConstraintScope: {
					&constraints.EdgeConstraint{
						Operator: constraints.MustContainConstraintOperator,
						Target: constraints.Edge{
							Source: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
							Target: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "orm"},
						},
						Node: construct.ResourceId{Provider: "mock", Type: "mock2", Name: "Corm"},
					},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"mock:mock1:mock1-compute",
					"mock:mock2:Corm",
					"mock:mock3:mock3-orm",
				},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock1:mock1-compute", Destination: "mock:mock2:Corm"},
					{Source: "mock:mock2:Corm", Destination: "mock:mock3:mock3-orm"},
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

			cg := construct.NewConstructGraph()
			for _, c := range tt.constructs {
				cg.AddConstruct(c)
			}
			for _, e := range tt.edges {
				cg.AddDependency(e.Source, e.Target)
			}

			engine.LoadContext(cg, tt.constraints, "test")
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument
			dag, err := engine.Run()
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
		})
	}
}
