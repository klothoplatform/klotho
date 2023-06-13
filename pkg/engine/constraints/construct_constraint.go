package constraints

import (
	"errors"

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
	switch b.Operator {
	case EqualsConstraintOperator:
		// Well look at all resources to see if there is a resource matching the type, that references the base construct passed in
		// Cuirrently attributes go unchecked
		if b.Type != "" {
			for _, res := range dag.ListResources() {
				if res.Id().Type == b.Type && res.BaseConstructsRef().HasId(b.Target) {
					return true
				}
			}
		}
	}
	return false
}

func (b *ConstructConstraint) Conflict(other Constraint) bool {
	return false
}

func (b *ConstructConstraint) Validate() error {
	if b.Target.Provider != core.AbstractConstructProvider {
		return errors.New("node constraint must be applied to an abstract construct")
	}
	return nil
}
