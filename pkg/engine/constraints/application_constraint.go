package constraints

import "github.com/klothoplatform/klotho/pkg/core"

type (
	// ApplicationConstraint is a struct that represents constraints that can be applied on the entire resource graph
	ApplicationConstraint struct {
		Operator        ConstraintOperator `yaml:"operator"`
		Node            core.ResourceId    `yaml:"node"`
		ReplacementNode core.ResourceId    `yaml:"replacement_node"`
		Edge            Edge               `yaml:"edge"`
	}
)

func (b *ApplicationConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (b *ApplicationConstraint) IsSatisfied(dag *core.ResourceGraph) bool {
	return false
}

func (b *ApplicationConstraint) Apply(dag *core.ResourceGraph) error {
	return nil
}

func (b *ApplicationConstraint) Conflict(other Constraint) bool {
	return false
}
