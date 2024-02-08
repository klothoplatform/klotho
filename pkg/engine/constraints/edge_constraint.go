package constraints

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// EdgeConstraint is a struct that represents constraints that can be applied on a specific edge in the resource graph
	//
	// Example
	//
	// To specify a constraint showing an edge must contain an intermediate node in its path, use the yaml below.
	//
	//- scope: edge
	//  operator: must_contain
	//  target:
	//    source: klotho:execution_unit:my_compute
	//    target: klotho:orm:my_orm
	//  node: aws:rds_proxy:my_proxy
	//
	// The end result of this should be a path of klotho:execution_unit:my_compute -> aws:rds_proxy:my_proxy -> klotho:orm:my_orm with N intermediate nodes to satisfy the path's expansion

	EdgeConstraint struct {
		Operator ConstraintOperator `yaml:"operator"`
		Target   Edge               `yaml:"target"`
	}
)

func (constraint *EdgeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (constraint *EdgeConstraint) IsSatisfied(ctx ConstraintGraph) bool {

	src := constraint.Target.Source
	dst := constraint.Target.Target
	// If we receive an abstract construct, we need to find all resources that reference the abstract construct
	//
	// This relies on resources only referencing an abstract provider if they are the direct child of the abstract construct
	// example
	// when we expand execution unit, the lambda would reference the execution unit as a construct, but the role and other resources would reference the lambda
	if constraint.Target.Source.IsAbstractResource() {
		srcRes := ctx.GetConstructsResource(constraint.Target.Source)
		if srcRes == nil {
			return false
		}
		src = srcRes.ID
	}

	if constraint.Target.Target.IsAbstractResource() {
		dstRes := ctx.GetConstructsResource(constraint.Target.Target)
		if dstRes == nil {
			return false
		}
		dst = dstRes.ID
	}

	paths, err := ctx.AllPaths(src, dst)
	if err != nil {
		return false
	}
	for _, path := range paths {
		if constraint.checkSatisfication(path, ctx) {
			return true
		}
	}
	return false
}

func (constraint *EdgeConstraint) checkSatisfication(path []*construct.Resource, ctx ConstraintGraph) bool {
	switch constraint.Operator {
	case MustExistConstraintOperator:
		return len(path) > 0
	case MustNotExistConstraintOperator:
		return len(path) == 0
	}
	return false
}

func (constraint *EdgeConstraint) Validate() error {
	if constraint.Target.Source == constraint.Target.Target {
		return fmt.Errorf("edge constraint must not have a source and target be the same node")
	}
	if (constraint.Target.Source == construct.ResourceId{} || constraint.Target.Target == construct.ResourceId{}) {
		return fmt.Errorf("edge constraint must have a source and target defined")
	}
	return nil
}

func (constraint *EdgeConstraint) String() string {
	return fmt.Sprintf("EdgeConstraint{Operator: %s, Target: %s}", constraint.Operator, constraint.Target)
}
