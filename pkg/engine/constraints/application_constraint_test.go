package constraints

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ApplicationConstraint_IsSatisfied(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "compute"}
	eu2 := &types.ExecutionUnit{Name: "compute2"}

	tests := []struct {
		name       string
		constraint []ApplicationConstraint
		resources  []construct.Resource
		want       bool
	}{
		{
			name: "Add is satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator: AddConstraintOperator,
					Node:     construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: AddConstraintOperator,
					Node:     construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []construct.Resource{
				&resources.LambdaFunction{
					Name: "my_function_also",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(eu),
				},
			},
			want: true,
		},
		{
			name: "Add is not satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator: AddConstraintOperator,
					Node:     construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: AddConstraintOperator,
					Node:     construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []construct.Resource{
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
					Node:     construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: RemoveConstraintOperator,
					Node:     construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []construct.Resource{
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
					Node:     construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
				},
				{
					Operator: RemoveConstraintOperator,
					Node:     construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
				},
			},
			resources: []construct.Resource{
				&resources.LambdaFunction{
					Name: "my_function_also",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(eu),
				},
			},
			want: false,
		},
		{
			name: "replace is satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator:        ReplaceConstraintOperator,
					Node:            construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute2"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "lambda_compute"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
					ReplacementNode: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also2"},
				},
			},
			resources: []construct.Resource{
				&resources.LambdaFunction{
					Name: "lambda_compute",
				},
				&resources.LambdaFunction{
					Name: "my_function_also2",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(eu2),
				},
			},
			want: true,
		},
		{
			name: "replace is not satisfied",
			constraint: []ApplicationConstraint{
				{
					Operator:        ReplaceConstraintOperator,
					Node:            construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute2"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "compute"},
					ReplacementNode: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "lambda_compute"},
				},
				{
					Operator:        ReplaceConstraintOperator,
					Node:            construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also"},
					ReplacementNode: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function_also2"},
				},
			},
			resources: []construct.Resource{
				&resources.LambdaFunction{
					Name: "my_function_also",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(eu),
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
			for _, constraint := range tt.constraint {
				result := constraint.IsSatisfied(dag, knowledgebase.EdgeKB{}, make(map[construct.ResourceId][]construct.Resource), nil)
				assert.Equalf(tt.want, result, "constraint %s is not satisfied", constraint)
			}
		})
	}
}
