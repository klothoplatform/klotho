package constraints

import (
	"errors"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
)

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
	switch b.Operator {
	case EqualsConstraintOperator:
		res := dag.GetResource(b.Target)
		if res == nil {
			return false
		}
		val := reflect.ValueOf(res).Elem().FieldByName(b.Property)
		return val.Interface() == b.Value
	}
	return false
}

func (b *NodeConstraint) Conflict(other Constraint) bool {
	return false
}

func (b *NodeConstraint) Validate() error {
	if b.Target.Provider == core.AbstractConstructProvider {
		return errors.New("node constraint cannot be applied to an abstract construct")
	}
	if b.Property == "" || reflect.ValueOf(b.Value).IsZero() {
		return errors.New("node constraint must have a property and value defined")
	}
	return nil
}
