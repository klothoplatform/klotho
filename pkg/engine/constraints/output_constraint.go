package constraints

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// OutputConstraint is a struct that represents a constraint exports some output from the resource graph
	//
	// Example
	//
	// To specify a constraint detailing application level intents in yaml
	//
	//- scope: output
	//  operator: add
	//  ref: aws:ec2:instance:my_instance#public_ip
	//  name: my_instance_public_ip
	//
	// The end result of this should be that the execution unit construct is added to the construct graph for processing
	OutputConstraint struct {
		Operator ConstraintOperator    `yaml:"operator" json:"operator"`
		Ref      construct.PropertyRef `yaml:"ref" json:"ref"`
		Name     string                `yaml:"name" json:"name"`
		Value    any                   `yaml:"value" json:"value"`
	}
)

func (constraint *OutputConstraint) Scope() ConstraintScope {
	return OutputConstraintScope
}

func (constraint *OutputConstraint) IsSatisfied(ctx ConstraintGraph) bool {
	return true
}

func (constraint *OutputConstraint) Validate() error {
	return nil
}

func (constraint *OutputConstraint) String() string {
	return fmt.Sprintf("OutputConstraint: %s %s %s", constraint.Operator, constraint.Name, constraint.Ref)
}
