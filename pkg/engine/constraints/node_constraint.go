package constraints

import "github.com/klothoplatform/klotho/pkg/core"

type (
	// NodeConstraint is a struct that represents constraints that can be applied on a specific node in the resource graph.
	// NodeConstraints are used to control intrinsic properties of a node in the resource graph
	NodeConstraint struct {
		Operator ConstraintOperator `yaml:"operator"`
		Target   core.ResourceId    `yaml:"target"`
		Property string             `yaml:"property"`
		Value    any                `yaml:"value"`
	}
)

func (b *NodeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (b *NodeConstraint) IsSatisfied(dag *core.ResourceGraph) bool {
	return false
}

func (b *NodeConstraint) Apply(dag *core.ResourceGraph) error {
	return nil
}

func (b *NodeConstraint) Conflict(other Constraint) bool {
	return false
}
