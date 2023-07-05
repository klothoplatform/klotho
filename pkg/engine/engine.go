package engine

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
	"go.uber.org/zap"
)

type (
	// Engine is a struct that represents the object which processes the resource graph and applies constraints
	Engine struct {
		// The providers that the engine is running against
		Providers map[string]provider.Provider
		// The knowledge base that the engine is running against
		KnowledgeBase knowledgebase.EdgeKB
		// The constructs which the engine understands
		Constructs []core.Construct
		// The context of the engine
		Context EngineContext
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Constraints                     map[constraints.ConstraintScope][]constraints.Constraint
		InitialState                    *core.ConstructGraph
		WorkingState                    *core.ConstructGraph
		EndState                        *core.ResourceGraph
		Decisions                       []Decision
		constructToResourceMapping      map[core.ResourceId][]core.Resource
		AppName                         string
		ExpandendOrCopiedBaseConstructs map[core.ResourceId]bool
		CopiedEdges                     map[core.ResourceId]map[core.ResourceId]bool
		ExpandedEdges                   map[core.ResourceId]map[core.ResourceId]bool
		ConfiguredEdges                 map[core.ResourceId]map[core.ResourceId]bool
		OperationalResources            map[core.ResourceId]bool
		ConfiguredResources             map[core.ResourceId]bool
		Errors                          map[int][]error
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

func NewEngine(providers map[string]provider.Provider, kb knowledgebase.EdgeKB, constructs []core.Construct) *Engine {
	return &Engine{
		Providers:     providers,
		KnowledgeBase: kb,
		Constructs:    constructs,
	}
}

func (e *Engine) LoadContext(initialState *core.ConstructGraph, constraints map[constraints.ConstraintScope][]constraints.Constraint, appName string) {
	e.Context = EngineContext{
		InitialState:                    initialState,
		Constraints:                     constraints,
		WorkingState:                    initialState.Clone(),
		EndState:                        core.NewResourceGraph(),
		constructToResourceMapping:      make(map[core.ResourceId][]core.Resource),
		AppName:                         appName,
		ExpandendOrCopiedBaseConstructs: make(map[core.ResourceId]bool),
		CopiedEdges:                     make(map[core.ResourceId]map[core.ResourceId]bool),
		ExpandedEdges:                   make(map[core.ResourceId]map[core.ResourceId]bool),
		ConfiguredEdges:                 make(map[core.ResourceId]map[core.ResourceId]bool),
		OperationalResources:            make(map[core.ResourceId]bool),
		ConfiguredResources:             make(map[core.ResourceId]bool),
		Errors:                          make(map[int][]error),
	}
}

// Run invokes the engine workflow to translate the initial state construct graph into the end state resource graph
//
// The steps of the engine workflow are
// - Apply all application constraints
// - Apply all edge constraints
// - Expand all constructs in the working state using the engines provider
// - Copy all dependencies from the working state to the end state
// - Apply all failed edge constraints
// - Expand all edges in the end state using the engines knowledge base and the EdgeConstraints provided
// - Configure all resources by applying ResourceConstraints
// - Configure all resources in the end state using the engines knowledge base
func (e *Engine) Run() (*core.ResourceGraph, error) {

	appliedConstraints := map[constraints.ConstraintScope]map[constraints.Constraint]bool{
		constraints.ApplicationConstraintScope: make(map[constraints.Constraint]bool),
		constraints.EdgeConstraintScope:        make(map[constraints.Constraint]bool),
	}

	NUM_LOOPS := 5

	for i := 0; i < NUM_LOOPS; i++ {
		zap.S().Debugf("Applying constraints iteration %d", i)
		// First we look at all application constraints to see what is going to be added and removed from the construct graph
		for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
			if !appliedConstraints[constraints.ApplicationConstraintScope][constraint] {
				err := e.ApplyApplicationConstraint(constraint.(*constraints.ApplicationConstraint))
				if err == nil {
					appliedConstraints[constraints.ApplicationConstraintScope][constraint] = true
				} else {
					e.Context.Errors[i] = append(e.Context.Errors[i], err)
				}
			}

		}

		// These edge constraints are at a construct level
		for _, constraint := range e.Context.Constraints[constraints.EdgeConstraintScope] {
			if !appliedConstraints[constraints.EdgeConstraintScope][constraint] {
				err := e.ApplyEdgeConstraint(constraint.(*constraints.EdgeConstraint))
				if err == nil {
					appliedConstraints[constraints.EdgeConstraintScope][constraint] = true
				} else {
					e.Context.Errors[i] = append(e.Context.Errors[i], err)
				}
			}
		}

		zap.S().Debug("Engine Expanding constructs and copying edges")
		err := e.ExpandConstructsAndCopyEdges()
		if err != nil {
			e.Context.Errors[i] = append(e.Context.Errors[i], err)
		}
		zap.S().Debug("Engine done Expanding constructs and copying edges")

		zap.S().Debug("Engine Expanding constructs and copying edges")
		// These edge constraints are at a resource level and must be applied before we expand edges otherwise we risk not satisfying constraints
		for _, constraint := range e.Context.Constraints[constraints.EdgeConstraintScope] {
			if !appliedConstraints[constraints.EdgeConstraintScope][constraint] {
				err := e.ApplyEdgeConstraint(constraint.(*constraints.EdgeConstraint))
				if err == nil {
					appliedConstraints[constraints.EdgeConstraintScope][constraint] = true
				} else {
					e.Context.Errors[i] = append(e.Context.Errors[i], err)
				}
			}
		}

		zap.S().Debug("Engine Expanding Edges")
		for _, dep := range e.Context.EndState.ListDependencies() {
			src := dep.Source.Id()
			dst := dep.Destination.Id()
			if e.Context.ExpandedEdges[src] == nil {
				e.Context.ExpandedEdges[src] = make(map[core.ResourceId]bool)
			}
			// If we know that the edge has a direct connection but is flipped due to data flow, immediately use that edge
			if det, _ := e.KnowledgeBase.GetEdge(dep.Source, dep.Destination); det.ReverseDirection {
				dep = graph.Edge[core.Resource]{Source: dep.Destination, Destination: dep.Source}
			}
			if !e.Context.ExpandedEdges[src][dst] {
				err = e.KnowledgeBase.ExpandEdge(&dep, e.Context.EndState)
				if err != nil {
					zap.S().Warnf("got error when expanding edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
					e.Context.Errors[i] = append(e.Context.Errors[i], err)
					continue
				}
			}
			e.Context.ExpandedEdges[src][dst] = true
		}
		zap.S().Debug("Engine Done Expanding Edges")
		zap.S().Debug("Engine configuring edges")
		for _, dep := range e.Context.EndState.ListDependencies() {
			if e.Context.ConfiguredEdges[dep.Source.Id()] != nil && e.Context.ConfiguredEdges[dep.Source.Id()][dep.Destination.Id()] {
				continue
			}
			err = e.KnowledgeBase.ConfigureEdge(&dep, e.Context.EndState)
			if err != nil {
				e.Context.Errors[i] = append(e.Context.Errors[i], err)
				continue
			}
			if e.Context.ConfiguredEdges[dep.Source.Id()] == nil {
				e.Context.ConfiguredEdges[dep.Source.Id()] = make(map[core.ResourceId]bool)
			}
			e.Context.ConfiguredEdges[dep.Source.Id()][dep.Destination.Id()] = true
		}
		zap.S().Debug("Engine done configuring edges")
		zap.S().Debug("Engine Making resources operational and configuring resources")
		for _, resource := range e.Context.EndState.ListResources() {
			err := e.Context.EndState.CallMakeOperational(resource, e.Context.AppName)
			if err != nil {
				if ore, ok := err.(*core.OperationalResourceError); ok {
					// If we get a OperationalResourceError let the engine try to reconcile it, and if that fails then mark the resource as non operational so we attempt to rerun on the next loop
					herr := e.handleOperationalResourceError(ore, e.Context.EndState)
					if herr != nil {
						err = errors.Join(err, herr)
					}
					e.Context.Errors[i] = append(e.Context.Errors[i], err)
					e.Context.OperationalResources[resource.Id()] = false
				}
				continue
			}
			e.Context.OperationalResources[resource.Id()] = true

			if !e.Context.ConfiguredResources[resource.Id()] {
				err := e.Context.EndState.CallConfigure(resource, nil)
				if err != nil {
					e.Context.Errors[i] = append(e.Context.Errors[i], err)
					continue
				}
				e.Context.ConfiguredResources[resource.Id()] = true
			}
		}
		zap.S().Debug("Engine done making resources operational and configuring resources")
		zap.S().Debug("Validating constraints")
		unsatisfiedConstraints := e.ValidateConstraints()

		if len(unsatisfiedConstraints) > 0 && i == NUM_LOOPS-1 {
			constraintsString := ""
			for _, constraint := range unsatisfiedConstraints {
				constraintsString += fmt.Sprintf("%s\n", constraint)
			}
			zap.S().Debugf("unsatisfied constraints: %s", constraintsString)
			return e.Context.EndState, fmt.Errorf("unsatisfied constraints: %s", constraintsString)
		} else {
			if len(e.Context.Errors[i]) == 0 {
				break
			} else if i == NUM_LOOPS-1 {
				var joinedErr error
				for _, error := range e.Context.Errors[i] {
					joinedErr = errors.Join(joinedErr, error)
				}
				return e.Context.EndState, fmt.Errorf("found the following errors during graph solving: %s", joinedErr.Error())
			} else {
				var joinedErr error
				for _, error := range e.Context.Errors[i] {
					joinedErr = errors.Join(joinedErr, error)
				}
				zap.S().Debugf("got errors: %s", joinedErr.Error())
			}
		}
	}

	zap.S().Debug("Validated constraints")
	return e.Context.EndState, nil
}

// ExpandConstructsAndCopyEdges expands all constructs in the working state using the engines provider
//
// The resources that result from the expanded constructs are written to the engines resource graph
// All dependencies are copied over to the resource graph
// If a dependency in the working state included a construct, the engine copies the dependency to all directly linked resources
func (e *Engine) ExpandConstructsAndCopyEdges() error {
	var joinedErr error
	for _, res := range e.Context.WorkingState.ListConstructs() {
		if e.Context.ExpandendOrCopiedBaseConstructs[res.Id()] {
			continue
		}
		// If the res is a resource, copy it over directly, otherwise we need to expand it
		if res.Id().Provider == core.AbstractConstructProvider {
			zap.S().Debugf("Expanding construct %s", res.Id())
			construct, ok := res.(core.Construct)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to cast base construct %s to construct while expanding construct", res.Id()))
				continue
			}

			// We want to see if theres any constraint nodes before we expand so that the constraint is expanded corretly
			// right now we will just look at the first constraint for the construct
			// TODO: Combine all constraints when needed for expansion
			constructType := ""
			attributes := make(map[string]any)
			for _, constraint := range e.Context.Constraints[constraints.ConstructConstraintScope] {
				constructConstraint, ok := constraint.(*constraints.ConstructConstraint)
				if !ok {
					joinedErr = errors.Join(joinedErr, fmt.Errorf(" constraint %s is incorrect type. Expected to be a construct constraint while expanding construct", constraint))
					continue
				}

				if constructConstraint.Target == construct.Id() {
					constructType = constructConstraint.Type
					attributes = constructConstraint.Attributes
					break
				}
			}
			var expandError error
			for _, provider := range e.Providers {
				mappedResources, err := provider.ExpandConstruct(construct, e.Context.WorkingState, e.Context.EndState, constructType, attributes)
				if err == nil {
					e.Context.constructToResourceMapping[res.Id()] = append(e.Context.constructToResourceMapping[res.Id()], mappedResources...)
					expandError = nil
					break
				} else {
					expandError = errors.Join(joinedErr, fmt.Errorf("unable to expand construct %s, %s", res.Id(), err.Error()))
				}

			}
			if expandError != nil {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to expand construct %s, %s", res.Id(), expandError.Error()))
			}
		} else {
			zap.S().Debugf("Copying resource over %s", res.Id())
			resource, ok := res.(core.Resource)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to cast base construct %s to resource while copying over resource", res.Id()))
				continue
			}
			e.Context.EndState.AddResource(resource)
		}
		e.Context.ExpandendOrCopiedBaseConstructs[res.Id()] = true
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
				if e.Context.CopiedEdges[srcNode.Id()] == nil {
					e.Context.CopiedEdges[srcNode.Id()] = make(map[core.ResourceId]bool)
				}
				if e.Context.CopiedEdges[srcNode.Id()][dstNode.Id()] {
					continue
				}

				zap.S().Debugf("Copying dependency %s -> %s", srcNode.Id(), dstNode.Id())
				e.Context.EndState.AddDependency(srcNode, dstNode)
				e.Context.CopiedEdges[srcNode.Id()][dstNode.Id()] = true
			}
		}
	}
	return joinedErr
}

// ApplyApplicationConstraint applies an application constraint to the either the engines working state construct graph
//
// Currently ApplicationConstraints can only be applied if the representing nodes are klotho constructs and not provider level resources
func (e *Engine) ApplyApplicationConstraint(constraint *constraints.ApplicationConstraint) error {
	decision := Decision{
		Constraint: constraint,
	}
	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		if constraint.Node.Provider == core.AbstractConstructProvider {
			construct, err := e.getConstructFromInputId(constraint.Node)
			if err != nil {
				return err
			}
			e.Context.WorkingState.AddConstruct(construct)
			decision.Construct = construct
		} else {
			provider := e.Providers[constraint.Node.Provider]
			resource, err := provider.CreateResourceFromId(constraint.Node, e.Context.InitialState)
			if err != nil {
				return err
			}
			e.Context.WorkingState.AddConstruct(resource)
		}
	case constraints.RemoveConstraintOperator:
		if constraint.Node.Provider == core.AbstractConstructProvider {
			construct := e.Context.WorkingState.GetConstruct(constraint.Node)
			if construct == nil {
				return fmt.Errorf("construct, %s, does not exist", constraint.Node)
			}
			decision.Construct = construct
			return e.Context.WorkingState.RemoveConstructAndEdges(construct)
		} else {
			resource := e.Context.EndState.GetResource(constraint.Node)
			if resource == nil {
				return fmt.Errorf("resource, %s, does not exist", constraint.Node)
			}
			decision.Resources = append(decision.Resources, resource)
			if !e.deleteResource(resource, true, true) {
				return fmt.Errorf("cannot remove resource %s, failed", constraint.Node)
			}
			return nil
		}
	case constraints.ReplaceConstraintOperator:
		if constraint.Node.Provider == core.AbstractConstructProvider {
			construct := e.Context.WorkingState.GetConstruct(constraint.Node)
			if construct == nil {
				return fmt.Errorf("construct, %s, does not exist", construct.Id())
			}
			new, err := e.getConstructFromInputId(constraint.ReplacementNode)
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

// ApplyEdgeConstraint applies an edge constraint to the either the engines working state construct graph or end state resource graph
//
// The following actions are taken for each operator
// - MustExistConstraintOperator, the edge is added to the working state construct graph
// - MustNotExistConstraintOperator, the edge is removed from the working state construct graph if the source and targets refer to klotho constructs. Otherwise the action fails
// - MustContainConstraintOperator, the constraint is applied to the edge before edge expansion, so when we use the knowledgebase to expand it ensures the node in the constraint is present in the expanded path
// - MustNotContainConstraintOperator, the constraint is applied to the edge before edge expansion, so when we use the knowledgebase to expand it ensures the node in the constraint is not present in the expanded path
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

		err := e.handleEdgeConstainConstraint(constraint)
		if err != nil {
			return err
		}

	case constraints.MustNotContainConstraintOperator:
		err := e.handleEdgeConstainConstraint(constraint)
		if err != nil {
			return err
		}
	}
	e.Context.Decisions = append(e.Context.Decisions, decision)
	return nil
}

// ApplyResourceConstraint applies a resource constraint to the end state resource graph
func (e *Engine) handleEdgeConstainConstraint(constraint *constraints.EdgeConstraint) error {
	srcNodes := []core.Resource{}
	dstNodes := []core.Resource{}

	if constraint.Target.Source.Provider == core.AbstractConstructProvider {
		srcResources, ok := e.Context.constructToResourceMapping[constraint.Target.Source]
		if !ok {
			return fmt.Errorf("unable to find resources for src construct %s needed to add edge data", constraint.Target.Source)
		}
		srcNodes = append(srcNodes, srcResources...)
	} else {
		src := e.Context.EndState.GetResource(constraint.Target.Source)
		if src == nil {
			return fmt.Errorf("unable to find resource %s", constraint.Target.Source)
		}
		srcNodes = append(srcNodes, src)
	}

	if constraint.Target.Target.Provider == core.AbstractConstructProvider {
		dstResources, ok := e.Context.constructToResourceMapping[constraint.Target.Target]
		if !ok {
			return fmt.Errorf("unable to find resources for dst construct %s needed to add edge data", constraint.Target.Target)
		}
		dstNodes = append(dstNodes, dstResources...)
	} else {
		dst := e.Context.EndState.GetResource(constraint.Target.Target)
		if dst == nil {
			return fmt.Errorf("unable to find resource %s", constraint.Target.Target)
		}
		dstNodes = append(dstNodes, dst)
	}
	provider := e.Providers[constraint.Node.Provider]
	resource, err := provider.CreateResourceFromId(constraint.Node, e.Context.WorkingState)
	if err != nil {
		return err
	}
	for _, src := range srcNodes {
		for _, dst := range dstNodes {

			var data knowledgebase.EdgeData
			dep := e.Context.EndState.GetDependency(constraint.Target.Source, constraint.Target.Target)
			if dep == nil {
				if constraint.Operator == constraints.MustContainConstraintOperator {
					data = knowledgebase.EdgeData{
						Constraint: knowledgebase.EdgeConstraint{
							NodeMustExist: []core.Resource{resource},
						},
					}
				} else if constraint.Operator == constraints.MustNotContainConstraintOperator {
					data = knowledgebase.EdgeData{
						Constraint: knowledgebase.EdgeConstraint{
							NodeMustNotExist: []core.Resource{resource},
						},
					}
				}
			} else {
				var ok bool
				data, ok = dep.Properties.Data.(knowledgebase.EdgeData)
				if dep.Properties.Data == nil {
					data = knowledgebase.EdgeData{}
				} else if !ok {
					return fmt.Errorf("unable to cast edge data for dep %s -> %s", constraint.Target.Source, constraint.Target.Target)
				}
				if constraint.Operator == constraints.MustContainConstraintOperator {
					data.Constraint.NodeMustExist = append(data.Constraint.NodeMustExist, resource)
				} else if constraint.Operator == constraints.MustNotContainConstraintOperator {
					data.Constraint.NodeMustNotExist = append(data.Constraint.NodeMustNotExist, resource)
				}
			}
			zap.S().Debugf("Adding edge data %v for %s -> %s", data, src.Id(), dst.Id())
			e.Context.EndState.AddDependencyWithData(src, dst, data)
		}
	}
	return nil
}

// ValidateConstraints validates all constraints against the end state resource graph
// It returns any constraints which were not satisfied by resource graphs current state
func (e *Engine) ValidateConstraints() []constraints.Constraint {
	var unsatisfied []constraints.Constraint
	for _, contextConstraints := range e.Context.Constraints {
		for _, constraint := range contextConstraints {
			if !constraint.IsSatisfied(e.Context.EndState, e.KnowledgeBase, e.Context.constructToResourceMapping) {
				unsatisfied = append(unsatisfied, constraint)
			}
		}

	}
	return unsatisfied
}
