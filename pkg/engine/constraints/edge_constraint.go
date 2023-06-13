package constraints

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
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
		Operator ConstraintOperator `yaml:"operator"`
		Target   Edge               `yaml:"target"`
		Node     core.ResourceId    `yaml:"node"`
	}
)

func (constraint *EdgeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

func (constraint *EdgeConstraint) IsSatisfied(dag *core.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[core.ResourceId][]core.Resource) bool {

	var src []core.ResourceId
	var dst []core.ResourceId
	// If we receive an abstract construct, we need to find all resources that reference the abstract construct
	//
	// This relies on resources only referencing an abstract provider if they are the direct child of the abstract construct
	// example
	// when we expand execution unit, the lambda would reference the execution unit as a construct, but the role and other resources would reference the lambda
	if constraint.Target.Source.Provider == core.AbstractConstructProvider {
		if len(mappedConstructResources[constraint.Target.Source]) == 0 {
			return false
		}
		for _, res := range mappedConstructResources[constraint.Target.Source] {
			src = append(src, res.Id())
		}
		src = append(src, mappedConstructResources[constraint.Target.Source]...)
	} else {
		src = append(src, constraint.Target.Source)
	}

	if constraint.Target.Target.Provider == core.AbstractConstructProvider {
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
			path, err := dag.ShortestPath(s, d)
			if err != nil {
				return false
			}
			// If theres no path in the dag we need to determine if there is a path due to inverted edges in the kb being used by the engine
			if len(path) == 0 {
				srcRes := dag.GetResource(s)
				if srcRes == nil {
					return false
				}
				dstRes := dag.GetResource(d)
				if dstRes == nil {
					return false
				}
				paths := kb.FindPathsInGraph(srcRes, dstRes, dag)
				if len(paths) == 0 {
					paths = append(paths, []graph.Edge[core.Resource]{})
				}

				for _, p := range paths {
					path := []core.Resource{}
					for _, res := range p {
						path = append(path, res.Source)
						path = append(path, res.Destination)
					}
					if constraint.checkSatisfication(path) {
						return true
					}
				}
				return false
			} else {
				if !constraint.checkSatisfication(path) {
					return false
				}
			}
		}
	}
	return true
}

func (constraint *EdgeConstraint) checkSatisfication(path []core.Resource) bool {
	// Currently we only support searching for if the node exists in the shortest path
	// We will likely want to search all paths to see if ANY contain the node. There's an open issue for this https://githuconstraint.com/dominikbraun/graph/issues/82
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
	if (constraint.Target.Source == core.ResourceId{} || constraint.Target.Target == core.ResourceId{}) {
		return fmt.Errorf("edge constraint must have a source and target defined")
	}
	if (constraint.Node == core.ResourceId{}) {
		return fmt.Errorf("edge constraint must have a node defined")
	}
	return nil
}
