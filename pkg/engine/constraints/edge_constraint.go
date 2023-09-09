package constraints

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
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
		Operator   ConstraintOperator   `yaml:"operator"`
		Target     Edge                 `yaml:"target"`
		Node       construct.ResourceId `yaml:"node"`
		Attributes map[string]any       `yaml:"attributes"`
	}
)

func (constraint *EdgeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (constraint *EdgeConstraint) IsSatisfied(dag *construct.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[construct.ResourceId][]construct.Resource, classifier *classification.ClassificationDocument) bool {

	var src []construct.ResourceId
	var dst []construct.ResourceId
	// If we receive an abstract construct, we need to find all resources that reference the abstract construct
	//
	// This relies on resources only referencing an abstract provider if they are the direct child of the abstract construct
	// example
	// when we expand execution unit, the lambda would reference the execution unit as a construct, but the role and other resources would reference the lambda
	if constraint.Target.Source.Provider == construct.AbstractConstructProvider {
		if len(mappedConstructResources[constraint.Target.Source]) == 0 {
			return false
		}
		for _, res := range mappedConstructResources[constraint.Target.Source] {
			src = append(src, res.Id())
		}
	} else {
		src = append(src, constraint.Target.Source)
	}

	if constraint.Target.Target.Provider == construct.AbstractConstructProvider {
		if len(mappedConstructResources[constraint.Target.Target]) == 0 {
			return false
		}
		for _, res := range mappedConstructResources[constraint.Target.Target] {
			dst = append(dst, res.Id())
		}
	} else {
		dst = append(dst, constraint.Target.Target)
	}
	for _, s := range src {
		for _, d := range dst {
			paths, err := dag.AllPaths(s, d)
			if err != nil {
				return false
			}
			if len(paths) == 0 {
				paths = append(paths, []construct.Resource{})
			}
			for _, path := range paths {
				if constraint.checkSatisfication(path, classifier) {
					return true
				}
			}
		}
	}
	return false
}

func (constraint *EdgeConstraint) checkSatisfication(path []construct.Resource, classifier *classification.ClassificationDocument) bool {
	if constraint.Attributes != nil {
		for i, res := range path {
			for k := range constraint.Attributes {
				if len(path) == 2 {
					if !collectionutil.Contains(classifier.GetClassification(path[0].Id()).Is, k) || !collectionutil.Contains(classifier.GetClassification(path[1].Id()).Is, k) {
						return false
					}
				} else {
					if !collectionutil.Contains(classifier.GetClassification(res.Id()).Is, k) && (i != 0 && i != len(path)-1) {
						return false
					}
				}
			}
		}
	}

	switch constraint.Operator {
	case MustContainConstraintOperator:
		for _, res := range path {
			if res.Id() == constraint.Node {
				return true
			}
		}
	case MustNotContainConstraintOperator:
		for _, res := range path {
			if res.Id() == constraint.Node {
				return false
			}
		}
		return true
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
	if (constraint.Node == construct.ResourceId{}) {
		return fmt.Errorf("edge constraint must have a node defined")
	}
	return nil
}

func (constraint *EdgeConstraint) String() string {
	return fmt.Sprintf("EdgeConstraint{Operator: %s, Target: %s, Node: %s, Attributes: %v}", constraint.Operator, constraint.Target, constraint.Node, constraint.Attributes)
}
