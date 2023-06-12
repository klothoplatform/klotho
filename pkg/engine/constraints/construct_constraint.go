package constraints

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	// ConstructConstraint is a struct that represents constraints that can be applied on a specific construct in the resource graph
	ConstructConstraint struct {
		Operator   ConstraintOperator `yaml:"operator"`
		Target     core.ResourceId    `yaml:"target"`
		Type       string             `yaml:"type"`
		Attributes map[string]any     `yaml:"attributes"`
	}
)

func (b *ConstructConstraint) Scope() ConstraintScope {
	return ConstructConstraintScope
}

func (b *ConstructConstraint) IsSatisfied(dag *core.ResourceGraph) bool {
	return false
}

func (b *ConstructConstraint) Apply(dag *core.ResourceGraph) error {
	return nil
}

func (b *ConstructConstraint) Conflict(other Constraint) bool {
	return false
}
