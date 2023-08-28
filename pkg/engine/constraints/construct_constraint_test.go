package constraints

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"

	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ConstructConstraint_IsSatisfied(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "compute"}
	tests := []struct {
		name       string
		constraint ConstructConstraint
		resources  []construct.Resource
		want       bool
	}{
		{
			name: "type equals is satisfied",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				Type:     "lambda_function",
			},
			resources: []construct.Resource{
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(eu),
				},
			},
			want: true,
		},
		{
			name: "type equals is not satisfied - wrong type",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				Type:     "lambda_function",
			},
			resources: []construct.Resource{
				&resources.Ec2Instance{
					Name:          "my_instance",
					ConstructRefs: construct.BaseConstructSetOf(eu),
				},
			},
			want: false,
		},
		{
			name: "type equals is not satisfied - no ref",
			constraint: ConstructConstraint{
				Operator: EqualsConstraintOperator,
				Target:   construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				Type:     "lambda_function",
			},
			resources: []construct.Resource{
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
				Target:   construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
			},
			resources: []construct.Resource{
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
			dag := construct.NewResourceGraph()
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			result := tt.constraint.IsSatisfied(dag, knowledgebase.EdgeKB{}, make(map[construct.ResourceId][]construct.Resource), nil)
			assert.Equal(tt.want, result)
		})
	}
}
