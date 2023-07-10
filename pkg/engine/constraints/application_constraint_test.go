package constraints

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ApplicationConstraint_IsSatisfied(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "compute"}
	eu2 := &core.ExecutionUnit{Name: "compute2"}

	tests := []struct {
		name       string
		constraint []ApplicationConstraint
		resources  []core.Resource
		want       bool
	}{
		{
			name: "Add is satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator: AddConstraintOperator,
					Node:     core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: AddConstraintOperator,
					Node:     core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function_also",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: core.BaseConstructSetOf(eu),
				},
			},
			want: true,
		},
		{
			name: "Add is not satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator: AddConstraintOperator,
					Node:     core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: AddConstraintOperator,
					Node:     core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function",
				},
			},
			want: false,
		},
		{
			name: "remove is satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator: RemoveConstraintOperator,
					Node:     core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: RemoveConstraintOperator,
					Node:     core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function",
				},
			},
			want: true,
		},
		{
			name: "remove is not satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator: RemoveConstraintOperator,
					Node:     core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: RemoveConstraintOperator,
					Node:     core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function_also",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: core.BaseConstructSetOf(eu),
				},
			},
			want: false,
		},
		{
			name: "replace is satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator:        ReplaceConstraintOperator,
					Node:            core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute2"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "lambda_compute"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
					ReplacementNode: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also2"},
				},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "lambda_compute",
				},
				&resources.LambdaFunction{
					Name: "my_function_also2",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: core.BaseConstructSetOf(eu2),
				},
			},
			want: true,
		},
		{
			name: "replace is not satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator:        ReplaceConstraintOperator,
					Node:            core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute2"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "lambda_compute"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
					ReplacementNode: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also2"},
				},
			},
			resources: []core.Resource{
				&resources.LambdaFunction{
					Name: "my_function_also",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: core.BaseConstructSetOf(eu),
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
			for _, constraint := range tt.constraint {
				result := constraint.IsSatisfied(dag, nil, make(map[core.ResourceId][]core.Resource))
				assert.Equalf(tt.want, result, "constraint %s is not satisfied", constraint)
			}
		})
	}
}
