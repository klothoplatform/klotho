package constraints

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"

	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_EdgeConstraint_IsSatisfied(t *testing.T) {
	tests := []struct {
		name            string
		constraint      EdgeConstraint
		resources       []construct.Resource
		edges           []Edge
		mappedResources map[construct.ResourceId][]construct.Resource
		want            bool
	}{
		{
			name: "must contain is satisfied - resource to resource",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must contain is satisfied - construct to resource",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			mappedResources: map[construct.ResourceId][]construct.Resource{
				{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"}: {
					&resources.LambdaFunction{Name: "my_function"},
				},
			},
			want: true,
		},
		{
			name: "must contain is satisfied - construct to construct",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
				&resources.RdsInstance{
					Name:          "my_instance",
					ConstructRefs: construct.BaseConstructSetOf(&types.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(&types.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			mappedResources: map[construct.ResourceId][]construct.Resource{
				{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"}: {
					&resources.LambdaFunction{Name: "my_function"},
				},
				{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "my_instance"}: {
					&resources.RdsInstance{Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must contain is satisfied - construct to construct - multiple constructs",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
				&resources.RdsInstance{
					Name: "my_instance",
				},
				&resources.LambdaFunction{
					Name: "my_function",
				},
				&resources.RdsInstance{
					Name: "my_instance2",
				},
				&resources.LambdaFunction{
					Name: "my_function2",
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function2"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance2"},
				},
			},
			mappedResources: map[construct.ResourceId][]construct.Resource{
				{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"}: {
					&resources.LambdaFunction{Name: "my_function"},
				},
				{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "my_instance"}: {
					&resources.RdsInstance{Name: "my_instance"},
				},
				{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"}: {
					&resources.LambdaFunction{Name: "my_function2"},
				},
				{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "my_instance"}: {
					&resources.RdsInstance{Name: "my_instance2"},
				},
			},
			want: true,
		},
		{
			name: "must contain is not satisfied",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "must contain is not satisfied - construct to construct - multiple constructs",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.EXECUTION_UNIT_TYPE, Name: "my_function"},
					Target: construct.ResourceId{Provider: construct.AbstractConstructProvider, Type: types.ORM_TYPE, Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
				&resources.RdsInstance{
					Name:          "my_instance",
					ConstructRefs: construct.BaseConstructSetOf(&types.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function",
					ConstructRefs: construct.BaseConstructSetOf(&types.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsInstance{
					Name:          "my_instance2",
					ConstructRefs: construct.BaseConstructSetOf(&types.Orm{Name: "my_instance"}),
				},
				&resources.LambdaFunction{
					Name:          "my_function2",
					ConstructRefs: construct.BaseConstructSetOf(&types.ExecutionUnit{Name: "my_function"}),
				},
				&resources.RdsProxy{
					Name: "my_proxy",
				},
			},
			edges: []Edge{
				{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},

			want: false,
		},
		{
			name: "must not contain is satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
		{
			name: "must not contain is not satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "no path between nodes in graph",
			constraint: EdgeConstraint{
				Operator: MustContainConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
				Node: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []construct.Resource{
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
			want: true,
		},
		{
			name: "must not exist not satisfied",
			constraint: EdgeConstraint{
				Operator: MustNotExistConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: false,
		},
		{
			name: "must exist not satisfied",
			constraint: EdgeConstraint{
				Operator: MustExistConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []construct.Resource{
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
			want: false,
		},
		{
			name: "must exist satisfied",
			constraint: EdgeConstraint{
				Operator: MustExistConstraintOperator,
				Target: Edge{
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			resources: []construct.Resource{
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
					Source: construct.ResourceId{Provider: "aws", Type: "lambda_function", Name: "my_function"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
				},
				{
					Source: construct.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					Target: construct.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := construct.NewResourceGraph()
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			for _, edge := range tt.edges {
				dag.AddDependencyById(edge.Source, edge.Target, nil)
			}
			result := tt.constraint.IsSatisfied(dag, knowledgebase.EdgeKB{}, tt.mappedResources, nil)
			assert.Equal(tt.want, result)
		})
	}
}
