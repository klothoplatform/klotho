package constraints

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	// EdgeConstraint is a struct that represents constraints that can be applied on a specific edge in the resource graph
	EdgeConstraint struct {
		Operator ConstraintOperator `yaml:"operator"`
		Target   Edge               `yaml:"target"`
		Node     core.ResourceId    `yaml:"node"`
	}
)

func (b *EdgeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (b *EdgeConstraint) IsSatisfied(dag *core.ResourceGraph) bool {

	var src []core.ResourceId
	var dst []core.ResourceId
	// If we receive an abstract construct, we need to find all resources that reference the abstract construct
	//
	// This relies on resources only referencing an abstract provider if they are the direct child of the abstract construct
	// example
	// when we expand execution unit, the lambda would reference the execution unit as a construct, but the role and other resources would reference the lambda
	if b.Target.Source.Provider == core.AbstractConstructProvider {
		for _, res := range dag.FindResourcesWithRef(b.Target.Source) {
			src = append(src, res.Id())
		}
	} else {
		src = append(src, b.Target.Source)
	}

	if b.Target.Target.Provider == core.AbstractConstructProvider {
		for _, res := range dag.FindResourcesWithRef(b.Target.Target) {
			dst = append(dst, res.Id())
		}
	} else {
		dst = append(dst, b.Target.Target)
	}

	for _, s := range src {
		for _, d := range dst {
			path, _ := dag.ShortestPath(s, d)
			if !b.checkSatisfication(path) {
				return false
			}
		}
	}
	return true
}

func (b *EdgeConstraint) checkSatisfication(path []core.Resource) bool {
	// Currently we only support MustContain & MustNotContainConstraintOperator searching for if the node exists in the shortest path
	// We will likely want to search all paths to see if ANY contain the node. There's an open issue for this https://github.com/dominikbraun/graph/issues/82
	switch b.Operator {
	case MustContainConstraintOperator:
		for _, res := range path {
			if res.Id() == b.Node {
				return true
			}
		}
	case MustNotContainConstraintOperator:
		for _, res := range path {
			if res.Id() == b.Node {
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

func (b *EdgeConstraint) Conflict(other Constraint) bool {
	return false
}

func (b *EdgeConstraint) Validate() error {
	if b.Target.Source == b.Target.Target {
		return fmt.Errorf("edge constraint must not have a source and target be the same node")
	}
	if (b.Target.Source == core.ResourceId{} || b.Target.Target == core.ResourceId{}) {
		return fmt.Errorf("edge constraint must have a source and target defined")
	}
	if (b.Node == core.ResourceId{}) {
		return fmt.Errorf("edge constraint must have a node defined")
	}
	return nil
}
