package constraints

import "github.com/klothoplatform/klotho/pkg/core"

type (
	// EdgeConstraint is a struct that represents constraints that can be applied on a specific edge in the resource graph
	EdgeConstraint struct {
		Operator ConstraintOperator `yaml:"operator"`
		Target   Edge               `yaml:"target"`
		Node     core.ResourceId    `yaml:"node"`
	}
)

func (b *EdgeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (b *EdgeConstraint) IsSatisfied(dag *core.ResourceGraph) bool {
	return false
}

func (b *EdgeConstraint) Apply(dag *core.ResourceGraph) error {
	return nil
}

func (b *EdgeConstraint) Conflict(other Constraint) bool {
	return false
}
