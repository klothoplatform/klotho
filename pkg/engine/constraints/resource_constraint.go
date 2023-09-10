package constraints

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type (
	// ResourceConstraint is a struct that represents constraints that can be applied on a specific node in the resource graph.
	// ResourceConstraints are used to control intrinsic properties of a resource in the resource graph
	//
	// Example
	//
	// To specify a constraint detailing a property of a resource in yaml
	//
	// - scope: resource
	// operator: equals
	// target: aws:rds_instance:my_instance
	// property: InstanceClass
	// value: db.t3.micro
	//
	// The end result of this should be that the the rds instance's InstanceClass property should be set to db.t3.micro
	ResourceConstraint struct {
		Operator ConstraintOperator   `yaml:"operator"`
		Target   construct.ResourceId `yaml:"target"`
		Property string               `yaml:"property"`
		Value    any                  `yaml:"value"`
	}
)

func (constraint *ResourceConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (constraint *ResourceConstraint) IsSatisfied(dag *construct.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[construct.ResourceId][]construct.Resource, classifier *classification.ClassificationDocument) bool {
	switch constraint.Operator {
	case EqualsConstraintOperator:
		res := dag.GetResource(constraint.Target)
		if res == nil {
			return false
		}
		val := reflect.ValueOf(res).Elem().FieldByName(constraint.Property)
		return val.IsValid() && val.Interface() == constraint.Value
	}
	return true
}

func (constraint *ResourceConstraint) Validate() error {
	if constraint.Target.Provider == construct.AbstractConstructProvider {
		return errors.New("node constraint cannot be applied to an abstract construct")
	}
	if constraint.Property == "" {
		return errors.New("node constraint must have a property defined")
	}
	return nil
}

func (constraint *ResourceConstraint) String() string {
	return fmt.Sprintf("ResourceConstraint: %v %v %v %v", constraint.Target, constraint.Property, constraint.Operator, constraint.Value)
}
