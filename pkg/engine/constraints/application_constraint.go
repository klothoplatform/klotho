package constraints

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	// ApplicationConstraint is a struct that represents constraints that can be applied on the entire resource graph
	ApplicationConstraint struct {
		Operator        ConstraintOperator `yaml:"operator"`
		Node            core.ResourceId    `yaml:"node"`
		ReplacementNode core.ResourceId    `yaml:"replacement_node"`
	}
)

func (b *ApplicationConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (b *ApplicationConstraint) IsSatisfied(dag *core.ResourceGraph) bool {
	switch b.Operator {
	case AddConstraintOperator:
		// If the add was for a construct, we need to check if any resource references the construct
		if b.Node.Provider == core.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(b.Node)) > 0
		}
		return dag.GetResource(b.Node) != nil
	case RemoveConstraintOperator:
		// If the remove was for a construct, we need to check if any resource references the construct
		if b.Node.Provider == core.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(b.Node)) == 0
		}
		return dag.GetResource(b.Node) == nil
	case ReplaceConstraintOperator:
		// We should entail edges are copied from the original source to the new replacement node in the dag
		// Ignoring for now, but will be an optimization we can make

		// If any of the nodes are abstract constructs, we need to check if any resource references the construct
		if b.Node.Provider == core.AbstractConstructProvider && b.ReplacementNode.Provider == core.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(b.Node)) == 0 && len(dag.FindResourcesWithRef(b.ReplacementNode)) > 0
		} else if b.Node.Provider == core.AbstractConstructProvider && b.ReplacementNode.Provider != core.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(b.Node)) == 0 && dag.GetResource(b.ReplacementNode) != nil
		} else if b.Node.Provider != core.AbstractConstructProvider && b.ReplacementNode.Provider == core.AbstractConstructProvider {
			return dag.GetResource(b.Node) == nil && len(dag.FindResourcesWithRef(b.Node)) > 0
		}
		return dag.GetResource(b.Node) == nil && dag.GetResource(b.ReplacementNode) != nil
	}
	return false
}

func (b *ApplicationConstraint) Conflict(other Constraint) bool {
	return false
}

func (b *ApplicationConstraint) Validate() error {
	if b.Operator == ReplaceConstraintOperator && (b.Node == core.ResourceId{} || b.ReplacementNode == core.ResourceId{}) {
		return errors.New("replace constraint must have a node and replacement node defined")
	}
	if b.Operator == ReplaceConstraintOperator && b.Node.Provider != core.AbstractConstructProvider && b.ReplacementNode.Provider == core.AbstractConstructProvider {
		return errors.New("replace constraint cannot replace a resource with an abstract construct")
	}
	if b.Operator == AddConstraintOperator && (b.Node == core.ResourceId{}) {
		return errors.New("add constraint must have a node defined")
	}

	if b.Operator == RemoveConstraintOperator && (b.Node == core.ResourceId{}) {
		return errors.New("remove constraint must have a node defined")
	}
	return nil
}
