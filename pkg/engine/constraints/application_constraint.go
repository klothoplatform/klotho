package constraints

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
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
		Operator        ConstraintOperator   `yaml:"operator"`
		Node            construct.ResourceId `yaml:"node"`
		ReplacementNode construct.ResourceId `yaml:"replacement_node"`
	}
)

func (constraint *ApplicationConstraint) Scope() ConstraintScope {
	return ApplicationConstraintScope
}

func (constraint *ApplicationConstraint) IsSatisfied(dag *construct.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[construct.ResourceId][]construct.Resource, classifier classification.Classifier) bool {
	switch constraint.Operator {
	case AddConstraintOperator:
		// If the add was for a construct, we need to check if any resource references the construct
		if constraint.Node.Provider == construct.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(constraint.Node)) > 0
		}
		return dag.GetResource(constraint.Node) != nil
	case RemoveConstraintOperator:
		// If the remove was for a construct, we need to check if any resource references the construct
		if constraint.Node.Provider == construct.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(constraint.Node)) == 0
		}
		return dag.GetResource(constraint.Node) == nil
	case ReplaceConstraintOperator:

		// We should ensure edges are copied from the original source to the new replacement node in the dag
		// Ignoring for now, but will be an extra check we can make later to ensure that the Replace constraint is fully satisfied

		// If any of the nodes are abstract constructs, we need to check if any resource references the construct
		if constraint.Node.Provider == construct.AbstractConstructProvider && constraint.ReplacementNode.Provider == construct.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(constraint.Node)) == 0 && len(dag.FindResourcesWithRef(constraint.ReplacementNode)) > 0
		} else if constraint.Node.Provider == construct.AbstractConstructProvider && constraint.ReplacementNode.Provider != construct.AbstractConstructProvider {
			return len(dag.FindResourcesWithRef(constraint.Node)) == 0 && dag.GetResource(constraint.ReplacementNode) != nil
		} else if constraint.Node.Provider != construct.AbstractConstructProvider && constraint.ReplacementNode.Provider == construct.AbstractConstructProvider {
			return dag.GetResource(constraint.Node) == nil && len(dag.FindResourcesWithRef(constraint.ReplacementNode)) > 0
		}
		return dag.GetResource(constraint.Node) == nil && dag.GetResource(constraint.ReplacementNode) != nil
	}
	return false
}

func (constraint *ApplicationConstraint) Validate() error {
	if constraint.Operator == ReplaceConstraintOperator && (constraint.Node == construct.ResourceId{} || constraint.ReplacementNode == construct.ResourceId{}) {
		return errors.New("replace constraint must have a node and replacement node defined")
	}
	if constraint.Operator == ReplaceConstraintOperator && constraint.Node.Provider != construct.AbstractConstructProvider && constraint.ReplacementNode.Provider == construct.AbstractConstructProvider {
		return errors.New("replace constraint cannot replace a resource with an abstract construct")
	}
	if constraint.Operator == AddConstraintOperator && (constraint.Node == construct.ResourceId{}) {
		return errors.New("add constraint must have a node defined")
	}

	if constraint.Operator == RemoveConstraintOperator && (constraint.Node == construct.ResourceId{}) {
		return errors.New("remove constraint must have a node defined")
	}
	return nil
}

func (constraint *ApplicationConstraint) String() string {
	return fmt.Sprintf("ApplicationConstraint: %s %s %s", constraint.Operator, constraint.Node, constraint.ReplacementNode)
}
