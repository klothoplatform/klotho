package engine

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
)

type (
	// Engine is a struct that represents the object which processes the resource graph and applies constraints
	Engine struct {
		// The provider that the engine is running against
		Provider provider.Provider
		// The knowledge base that the engine is running against
		KnowledgeBase knowledgebase.EdgeKB
		// The context of the engine
		Context EngineContext
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Constraints                map[constraints.ConstraintScope][]constraints.Constraint
		InitialState               *core.ConstructGraph
		WorkingState               *core.ConstructGraph
		EndState                   *core.ResourceGraph
		Decisions                  []Decision
		constructToResourceMapping map[core.ResourceId][]core.Resource
		AppName                    string
	}

	// Decision is a struct that represents a decision made by the engine
	Decision struct {
		// The resources that was modified
		Resources []core.Resource
		// The edges that were modified
		Edges []constraints.Edge
		// The constructs that influenced this if applicable
		Construct core.BaseConstruct
		// The constraint that was applied
		Constraint constraints.Constraint
	}
)

func NewEngine(provider provider.Provider, kb knowledgebase.EdgeKB) *Engine {
	return &Engine{
		Provider:      provider,
		KnowledgeBase: kb,
	}
}

func (e *Engine) LoadContext(initialState *core.ConstructGraph, constraints map[constraints.ConstraintScope][]constraints.Constraint, appName string) {
	e.Context = EngineContext{
		InitialState:               initialState,
		Constraints:                constraints,
		WorkingState:               initialState.Clone(),
		EndState:                   core.NewResourceGraph(),
		constructToResourceMapping: make(map[core.ResourceId][]core.Resource),
		AppName:                    appName,
	}
}

func (e *Engine) Run() (*core.ResourceGraph, error) {

	appliedConstraints := map[constraints.ConstraintScope]map[constraints.Constraint]bool{}

	// First we look at all application constraints to see what is going to be added and removed from the construct graph
	for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
		err := e.ApplyApplicationConstraint(constraint.(*constraints.ApplicationConstraint))
		if err == nil {
			appliedConstraints[constraints.ApplicationConstraintScope][constraint] = true
		}
	}

	// These edge constraints are at a construct level
	for _, constraint := range e.Context.Constraints[constraints.EdgeConstraintScope] {
		err := e.ApplyEdgeConstraint(constraint.(*constraints.EdgeConstraint))
		if err == nil {
			appliedConstraints[constraints.EdgeConstraintScope][constraint] = true
		}
	}

	err := e.ExpandConstructsAndCopyEdges()
	if err != nil {
		return nil, err
	}

	// Apply the remainder of the edge constraints after we have expanded our graph
	for _, constraint := range e.Context.Constraints[constraints.EdgeConstraintScope] {
		if applied := appliedConstraints[constraints.EdgeConstraintScope][constraint]; !applied {
			err := e.ApplyEdgeConstraint(constraint.(*constraints.EdgeConstraint))
			if err == nil {
				appliedConstraints[constraints.EdgeConstraintScope][constraint] = true
			}
		}
	}

	err = e.KnowledgeBase.ExpandEdges(e.Context.EndState, e.Context.AppName)
	if err != nil {
		return nil, err
	}

	err = e.KnowledgeBase.ConfigureFromEdgeData(e.Context.EndState)
	if err != nil {
		return e.Context.EndState, err
	}

	unsatisfiedConstraints := e.ValidateConstraints()

	if len(unsatisfiedConstraints) > 0 {
		return e.Context.EndState, fmt.Errorf("unsatisfied constraints: %v", unsatisfiedConstraints)
	}

	return e.Context.EndState, nil
}

func (e *Engine) ExpandConstructsAndCopyEdges() error {
	var joinedErr error
	for _, res := range e.Context.WorkingState.ListConstructs() {
		// If the res is a resource, copy it over directly, otherwise we need to expand it
		if res.Id().Provider != core.AbstractConstructProvider {
			construct, ok := res.(core.Construct)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to cast base construct %s to construct", res.Id()))
				continue
			}
			mappedResources, err := e.Provider.ExpandConstruct(construct, e.Context.EndState)
			if err != nil {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to expand construct %s", res.Id()))
			}
			e.Context.constructToResourceMapping[res.Id()] = append(e.Context.constructToResourceMapping[res.Id()], mappedResources...)
		} else {
			resource, ok := res.(core.Resource)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to cast base construct %s to construct", res.Id()))
				continue
			}
			e.Context.EndState.AddResource(resource)
		}
	}

	for _, dep := range e.Context.WorkingState.ListDependencies() {
		srcNodes := []core.Resource{}
		dstNodes := []core.Resource{}
		if dep.Source.Id().Provider == core.AbstractConstructProvider {
			srcResources, ok := e.Context.constructToResourceMapping[dep.Source.Id()]
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resources for construct %s", dep.Source.Id()))
				continue
			}
			srcNodes = append(srcNodes, srcResources...)
		} else {
			resource, ok := dep.Source.(core.Resource)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to cast base construct %s to resource", dep.Source.Id()))
				continue
			}
			srcNodes = append(srcNodes, resource)
		}

		if dep.Destination.Id().Provider == core.AbstractConstructProvider {
			dstResources, ok := e.Context.constructToResourceMapping[dep.Destination.Id()]
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resources for construct %s", dep.Destination.Id()))
				continue
			}
			dstNodes = append(dstNodes, dstResources...)
		} else {
			resource, ok := dep.Destination.(core.Resource)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to cast base construct %s to resource", dep.Destination.Id()))
				continue
			}
			dstNodes = append(dstNodes, resource)
		}

		for _, srcNode := range srcNodes {
			for _, dstNode := range dstNodes {
				e.Context.EndState.AddDependency(srcNode, dstNode)
			}
		}
	}
	return joinedErr
}

func (e *Engine) ApplyApplicationConstraint(constraint *constraints.ApplicationConstraint) error {
	decision := Decision{
		Constraint: constraint,
	}
	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		if constraint.Node.Provider == core.AbstractConstructProvider {
			construct, err := core.GetConstructFromInputId(constraint.Node)
			if err != nil {
				return err
			}
			e.Context.WorkingState.AddConstruct(construct)
			decision.Construct = construct
		}
	case constraints.RemoveConstraintOperator:
		if constraint.Node.Provider == core.AbstractConstructProvider {
			construct := e.Context.WorkingState.GetConstruct(constraint.Node)
			if construct == nil {
				return fmt.Errorf("construct, %s, does not exist", construct.Id())
			}
			decision.Construct = construct
			return e.Context.WorkingState.RemoveConstructAndEdges(construct)
		} else {
			return fmt.Errorf("cannot remove resource %s, removing resources is not supported at this time", constraint.Node)
		}
	case constraints.ReplaceConstraintOperator:
		if constraint.Node.Provider == core.AbstractConstructProvider {
			construct := e.Context.WorkingState.GetConstruct(constraint.Node)
			if construct == nil {
				return fmt.Errorf("construct, %s, does not exist", construct.Id())
			}
			new, err := core.GetConstructFromInputId(constraint.ReplacementNode)
			if err != nil {
				return err
			}
			decision.Construct = construct
			return e.Context.WorkingState.ReplaceConstruct(construct, new)
		} else {
			return fmt.Errorf("cannot replace resource %s, replacing resources is not supported at this time", constraint.Node)
		}
	}
	e.Context.Decisions = append(e.Context.Decisions, decision)
	return nil
}

func (e *Engine) ApplyEdgeConstraint(constraint *constraints.EdgeConstraint) error {
	decision := Decision{
		Constraint: constraint,
	}
	switch constraint.Operator {
	case constraints.MustExistConstraintOperator:
		e.Context.WorkingState.AddDependency(constraint.Target.Source, constraint.Target.Target)
	case constraints.MustNotExistConstraintOperator:
		if constraint.Target.Source.Provider == core.AbstractConstructProvider && constraint.Target.Target.Provider == core.AbstractConstructProvider {
			decision.Edges = []constraints.Edge{constraint.Target}
			return e.Context.WorkingState.RemoveDependency(constraint.Target.Source, constraint.Target.Target)
		} else {
			return fmt.Errorf("edge constraints with the MustNotExistConstraintOperator are not available at this time for resources, %s", constraint.Target)
		}
	case constraints.MustContainConstraintOperator:
		if constraint.Target.Source.Provider == core.AbstractConstructProvider || constraint.Target.Target.Provider == core.AbstractConstructProvider {
			return fmt.Errorf("edge constraints with the MustContainConstraintOperator are not available at this time for constructs, %s", constraint.Target)
		}
		resource, err := e.Provider.CreateResourceFromId(constraint.Node, e.Context.EndState)
		if err != nil {
			return err
		}
		var data knowledgebase.EdgeData
		dep := e.Context.EndState.GetDependency(constraint.Target.Source, constraint.Target.Target)
		if dep == nil {
			data = knowledgebase.EdgeData{
				Constraint: knowledgebase.EdgeConstraint{
					NodeMustExist: []core.Resource{resource},
				},
			}
		} else {
			var ok bool
			data, ok = dep.Properties.Data.(knowledgebase.EdgeData)
			if !ok {
				return fmt.Errorf("unable to cast edge data for dep %s -> %s", constraint.Target.Source, constraint.Target.Target)
			}
			data.Constraint.NodeMustExist = append(data.Constraint.NodeMustExist, resource)
		}

		src := e.Context.EndState.GetResource(constraint.Target.Source)
		if src == nil {
			return fmt.Errorf("unable to find resource %s", constraint.Target.Source)
		}
		dst := e.Context.EndState.GetResource(constraint.Target.Target)
		if dst == nil {
			return fmt.Errorf("unable to find resource %s", constraint.Target.Target)
		}
		e.Context.EndState.AddDependencyWithData(src, dst, data)
		return nil
	case constraints.MustNotContainConstraintOperator:
		if constraint.Target.Source.Provider == core.AbstractConstructProvider || constraint.Target.Target.Provider == core.AbstractConstructProvider {
			return fmt.Errorf("edge constraints with the MustContainConstraintOperator are not available at this time for constructs, %s", constraint.Target)
		}
		resource, err := e.Provider.CreateResourceFromId(constraint.Node, e.Context.EndState)
		if err != nil {
			return err
		}
		var data knowledgebase.EdgeData
		dep := e.Context.EndState.GetDependency(constraint.Target.Source, constraint.Target.Target)
		if dep == nil {
			data = knowledgebase.EdgeData{
				Constraint: knowledgebase.EdgeConstraint{
					NodeMustNotExist: []core.Resource{resource},
				},
			}
		} else {
			var ok bool
			data, ok = dep.Properties.Data.(knowledgebase.EdgeData)
			if !ok {
				return fmt.Errorf("unable to cast edge data for dep %s -> %s", constraint.Target.Source, constraint.Target.Target)
			}
			data.Constraint.NodeMustNotExist = append(data.Constraint.NodeMustNotExist, resource)
		}

		src := e.Context.EndState.GetResource(constraint.Target.Source)
		if src == nil {
			return fmt.Errorf("unable to find resource %s", constraint.Target.Source)
		}
		dst := e.Context.EndState.GetResource(constraint.Target.Target)
		if dst == nil {
			return fmt.Errorf("unable to find resource %s", constraint.Target.Target)
		}
		e.Context.EndState.AddDependencyWithData(src, dst, data)
		return nil
	}
	e.Context.Decisions = append(e.Context.Decisions, decision)
	return nil
}

func (e *Engine) ValidateConstraints() []constraints.Constraint {
	var unsatisfied []constraints.Constraint
	for _, contextConstraints := range e.Context.Constraints {
		for _, constraint := range contextConstraints {
			if !constraint.IsSatisfied(e.Context.EndState, e.Context.constructToResourceMapping) {
				unsatisfied = append(unsatisfied, constraint)
			}
		}

	}
	return unsatisfied
}

// func (e *Engine) RemoveResourceReconciliation(node core.ResourceId) error {
// 	resource := e.Context.WorkingState.GetConstruct(node)
// 	resource, ok := resource.()
// 	// in this context src is the node passed in
// 	for _, edge := range e.Context.WorkingState.GetDownstreamDependencies(resource) {

// 		dstResource, ok := edge.Destination.(core.Resource)

// 			// Since its a construct we just assume every single edge can be removed
// 		if !ok {
// 			e.Context.WorkingState.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
// 		}

// 		// If the edge is invalid, what should we do? Not solving yet
// 		kbEdge, found := e.KnowledgeBase.GetEdge(edge.Source, dstResource)
// 		if !found {
// 			e.Context.WorkingState.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
// 		}
// 		var eventualTarget core.BaseConstruct

// 		err := e.Context.WorkingState.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	for _, edge := range e.Context.WorkingState.GetUpstreamDependencies(resource) {
// 		err := e.Context.WorkingState.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return e.Context.WorkingState.RemoveConstruct(resource)
// }
