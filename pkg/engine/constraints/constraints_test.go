package constraints

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_ParseConstraintsFromFile(t *testing.T) {
	tests := []struct {
		name string
		file []byte
		want map[ConstraintScope][]Constraint
	}{
		{
			name: "test",
			file: []byte(`- scope: application
  operator: add
  node: klotho:execution_unit:my_compute
- scope: application
  operator: add
  node: klotho:orm:my_orm
- scope: construct
  operator: equals
  target: klotho:orm:my_orm
  type: rds_instance
- scope: edge
  operator: must_contain
  target: 
    source: klotho:execution_unit:my_compute
    target: klotho:orm:my_orm
  node: aws:rds_proxy:my_proxy
- scope: resource
  operator: add
  target: aws:rds_instance:my_instance
  property: db_instance_class
  value: db.t3.micro`),
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
				ResourceConstraintScope: {
					&ResourceConstraint{
						Operator: AddConstraintOperator,
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
			result, err := ParseConstraintsFromFile(tt.file)
			if !assert.NoError(err) {
				return
			}
			assert.ElementsMatch(tt.want[ApplicationConstraintScope], result[ApplicationConstraintScope])
			assert.ElementsMatch(tt.want[ConstructConstraintScope], result[ConstructConstraintScope])
			assert.ElementsMatch(tt.want[EdgeConstraintScope], result[EdgeConstraintScope])
		})
	}
}
