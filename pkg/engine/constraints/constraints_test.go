package constraints

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_ParseConstraintsFromFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want map[ConstraintScope][]Constraint
	}{
		{
			name: "test",
			path: "./samples/constraints.yaml",
			want: map[ConstraintScope][]Constraint{
				ApplicationConstraintScope: {
					&ApplicationConstraint{
						Operator: AddConstraintOperator,
						Node:     core.ResourceId{Provider: core.AbstractConstructProvider, Type: "execution_unit", Name: "my_compute"},
					},
					&ApplicationConstraint{
						Operator: AddConstraintOperator,
						Node:     core.ResourceId{Provider: core.AbstractConstructProvider, Type: "orm", Name: "my_orm"},
					},
				},
				ConstructConstraintScope: {
					&ConstructConstraint{
						Operator: EqualsConstraintOperator,
						Target:   core.ResourceId{Provider: core.AbstractConstructProvider, Type: "orm", Name: "my_orm"},
						Type:     "rds_instance",
					},
				},
				EdgeConstraintScope: {
					&EdgeConstraint{
						Operator: MustContainConstraintOperator,
						Target: Edge{
							Source: core.ResourceId{Provider: core.AbstractConstructProvider, Type: "execution_unit", Name: "my_compute"},
							Target: core.ResourceId{Provider: core.AbstractConstructProvider, Type: "orm", Name: "my_orm"},
						},
						Node: core.ResourceId{Provider: "aws", Type: "rds_proxy", Name: "my_proxy"},
					},
				},
				NodeConstraintScope: {
					&NodeConstraint{
						Operator: EqualsConstraintOperator,
						Target:   core.ResourceId{Provider: "aws", Type: "rds_instance", Name: "my_instance"},
						Property: "db_instance_class",
						Value:    "db.t3.micro",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result, err := ParseConstraintsFromFile(tt.path)
			if !assert.NoError(err) {
				return
			}
			assert.ElementsMatch(tt.want[ApplicationConstraintScope], result[ApplicationConstraintScope])
			assert.ElementsMatch(tt.want[ConstructConstraintScope], result[ConstructConstraintScope])
			assert.ElementsMatch(tt.want[EdgeConstraintScope], result[EdgeConstraintScope])
		})
	}
}
