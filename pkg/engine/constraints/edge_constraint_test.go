package constraints

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_EdgeConstraint_IsSatisfied(t *testing.T) {
	tests := []struct {
		name       string
		constraint EdgeConstraint
		resources  []core.Resource
		edges      []Edge
		want       bool
	}{
		{
			name: "must contain is satisfied - resource to resource",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must contain is satisfied - construct to resource",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructsRef: core.BaseConstructSetOf(&core.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must contain is satisfied - construct to construct",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name:          "my_instance",
					ConstructsRef: core.BaseConstructSetOf(&core.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructsRef: core.BaseConstructSetOf(&core.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must contain is satisfied - construct to construct - multiple constructs",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name:          "my_instance",
					ConstructsRef: core.BaseConstructSetOf(&core.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructsRef: core.BaseConstructSetOf(&core.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsInstance{
					Name:          "my_instance2",
					ConstructsRef: core.BaseConstructSetOf(&core.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function2",
					ConstructsRef: core.BaseConstructSetOf(&core.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function2"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance2"},
				},
			},
			want: true,
		},
		{
			name: "must contain is not satisfied",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "must contain is not satisfied - construct to construct - multiple constructs",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: core.ORM_TYPE, Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name:          "my_instance",
					ConstructsRef: core.BaseConstructSetOf(&core.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructsRef: core.BaseConstructSetOf(&core.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsInstance{
					Name:          "my_instance2",
					ConstructsRef: core.BaseConstructSetOf(&core.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function2",
					ConstructsRef: core.BaseConstructSetOf(&core.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "must not contain is satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must not contain is not satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "no path between nodes in graph",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{},
			want:  false,
		},
		{
			name: "must not exist satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotExistConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{},
			want:  true,
		},
		{
			name: "must not exist not satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotExistConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "must exist not satisfied",
			constraint: EdgeConstraint{
				Operator: MustExistConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{},
			want:  false,
		},
		{
			name: "must exist satisfied",
			constraint: EdgeConstraint{
				Operator: MustExistConstraintOperator,
				Target: Edge{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []core.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: core.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println(tt.name)
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			for _, edge := range tt.edges {
				dag.AddDependencyById(edge.Source, edge.Target, nil)
			}
			result := tt.constraint.IsSatisfied(dag)
			assert.Equal(tt.want, result)
		})
	}
}
