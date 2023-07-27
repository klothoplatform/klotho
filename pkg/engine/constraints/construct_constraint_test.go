package constraints

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ConstructConstraint_IsSatisfied(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "compute"}
	tests := []struct {
		name       string
		constraint ConstructConstraint
		resources  []core.Resource
		want       bool
	}{
		{
			name: "type equals is satisfied",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				Type:     "lambda_function",
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: core.BaseConstructSetOf(eu),
				},
			},
			want: true,
		},
		{
			name: "type equals is not satisfied - wrong type",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				Type:     "lambda_function",
			},
			resources: []core.Resource{
				&resources.Ec2Instance{
					Name:          "my_instance",
					ConstructRefs: core.BaseConstructSetOf(eu),
				},
			},
			want: false,
		},
		{
			name: "type equals is not satisfied - no ref",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				Type:     "lambda_function",
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function",
				},
			},
			want: false,
		},
		{
			name: "no equals is not satisfied and fails",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			result := tt.constraint.IsSatisfied(dag, knowledgebase.EdgeKB{}, make(map[core.ResourceId][]core.Resource), nil)
			assert.Equal(tt.want, result)
		})
	}
}
