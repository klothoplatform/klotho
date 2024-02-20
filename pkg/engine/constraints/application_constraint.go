package constraints

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// ApplicationConstraint is a struct that represents constraints that can be applied on the entire resource graph
	//
	// Example
	//
	// To specify a constraint detailing application level intents in yaml
	//
	//- scope: application
	//  operator: add
	//  node: klotho:execution_unit:my_compute
	//
	// The end result of this should be that the execution unit construct is added to the construct graph for processing
	ApplicationConstraint struct {
		Operator        ConstraintOperator   `yaml:"operator" json:"operator"`
		Node            construct.ResourceId `yaml:"node" json:"node"`
		ReplacementNode construct.ResourceId `yaml:"replacement_node,omitempty" json:"replacement_node,omitempty"`
	}
)

func (constraint *ApplicationConstraint) Scope() ConstraintScope {
	return ApplicationConstraintScope
}

func (constraint *ApplicationConstraint) IsSatisfied(ctx ConstraintGraph) bool {
	switch constraint.Operator {
	case AddConstraintOperator, MustExistConstraintOperator:
		nodeToSearchFor := constraint.Node
		// If the add was for a construct, we need to check if any resource references the construct
		if constraint.Node.IsAbstractResource() {
			nodeToSearchFor = ctx.GetConstructsResource(constraint.Node).ID
		}
		res, _ := ctx.GetResource(nodeToSearchFor)
		return res != nil

	case RemoveConstraintOperator, MustNotExistConstraintOperator:
		nodeToSearchFor := constraint.Node
		// If the remove was for a construct, we need to check if any resource references the construct
		if constraint.Node.IsAbstractResource() {
			nodeToSearchFor = ctx.GetConstructsResource(constraint.Node).ID
		}
		res, _ := ctx.GetResource(nodeToSearchFor)
		return res == nil
	case ReplaceConstraintOperator:

		// We should ensure edges are copied from the original source to the new replacement node in the dag
		// Ignoring for now, but will be an extra check we can make later to ensure that the Replace constraint is fully satisfied

		// If any of the nodes are abstract constructs, we need to check if any resource references the construct
		if constraint.Node.IsAbstractResource() && constraint.ReplacementNode.IsAbstractResource() {
			return ctx.GetConstructsResource(constraint.Node) == nil && ctx.GetConstructsResource(constraint.ReplacementNode) != nil
		} else if constraint.Node.IsAbstractResource() && !constraint.ReplacementNode.IsAbstractResource() {
			res, err := ctx.GetResource(constraint.ReplacementNode)
			if err != nil {
				return false
			}
			return ctx.GetConstructsResource(constraint.Node) == nil && res != nil
		} else if !constraint.Node.IsAbstractResource() && constraint.ReplacementNode.IsAbstractResource() {
			res, err := ctx.GetResource(constraint.Node)
			if err != nil {
				return false
			}
			return res == nil && ctx.GetConstructsResource(constraint.ReplacementNode) != nil
		}
		node, _ := ctx.GetResource(constraint.Node)
		replacementNode, _ := ctx.GetResource(constraint.ReplacementNode)
		return node == nil && replacementNode != nil
	}
	return false
}

func (constraint *ApplicationConstraint) Validate() error {
	switch constraint.Operator {
	case AddConstraintOperator, MustExistConstraintOperator:
		if constraint.Node.IsZero() {
			return errors.New("add/must_exist constraint must have a node defined")
		}

	case RemoveConstraintOperator, MustNotExistConstraintOperator:
		if constraint.Node.IsZero() {
			return errors.New("remove/must_not_exist constraint must have a node defined")
		}

	case ReplaceConstraintOperator:
		if constraint.Node.IsZero() || constraint.ReplacementNode.IsZero() {
			return errors.New("replace constraint must have a node and replacement node defined")
		}
	}
	return nil
}

func (constraint *ApplicationConstraint) String() string {
	return fmt.Sprintf("ApplicationConstraint: %s %s %s", constraint.Operator, constraint.Node, constraint.ReplacementNode)
}
