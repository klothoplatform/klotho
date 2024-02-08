package constraints

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// ConstructConstraint is a struct that represents constraints that can be applied on a specific construct in the resource graph
	//
	// Example
	//
	// To specify a constraint detailing construct expansion and configuration in yaml
	//
	// - scope: construct
	// operator: equals
	// target: klotho:orm:my_orm
	// type: rds_instance
	//
	// The end result of this should be that the orm construct is expanded into an rds instance + necessary resources
	ConstructConstraint struct {
		Operator   ConstraintOperator   `yaml:"operator"`
		Target     construct.ResourceId `yaml:"target"`
		Type       string               `yaml:"type"`
		Attributes map[string]any       `yaml:"attributes"`
	}
)

func (constraint *ConstructConstraint) Scope() ConstraintScope {
	return ConstructConstraintScope
}

func (constraint *ConstructConstraint) IsSatisfied(ctx ConstraintGraph) bool {
	switch constraint.Operator {
	case EqualsConstraintOperator:
		// Well look at all resources to see if there is a resource matching the type, that references the base construct passed in
		// Cuirrently attributes go unchecked
		res := ctx.GetConstructsResource(constraint.Target)
		if res == nil {
			return false
		}
		if constraint.Type != "" && res.ID.Type != constraint.Type {
			return false
		}
		return true
	}
	return false
}

func (constraint *ConstructConstraint) Validate() error {
	if !constraint.Target.IsAbstractResource() {
		return errors.New("node constraint must be applied to an abstract construct")
	}
	return nil
}

func (constraint *ConstructConstraint) String() string {
	return fmt.Sprintf("Constraint: %s %s %s", constraint.Scope(), constraint.Operator, constraint.Target)
}
