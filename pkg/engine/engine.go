package engine

import (
	"embed"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/input"
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
		// The classification document that the engine uses for understanding resources
		ClassificationDocument *classification.ClassificationDocument
		// The constructs which the engine understands
		Constructs []construct.Construct
		// The templates that the engine uses to make resources operational
		ResourceTemplates map[construct.ResourceId]*knowledgebase.ResourceTemplate
		// The templates that the engine uses to make edges operational
		EdgeTemplates map[string]*knowledgebase.EdgeTemplate
		// The context of the engine
		Context EngineContext

		Guardrails *Guardrails
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Input                       input.Input
		Constraints                 map[constraints.ConstraintScope][]constraints.Constraint
		InitialState                *construct.ConstructGraph
		WorkingState                *construct.ConstructGraph
		Solution                    *construct.ResourceGraph
		Decisions                   []Decision
		constructExpansionSolutions map[construct.ResourceId][]*ExpansionSolution
	}

	SolveContext struct {
		ResourceGraph       *construct.ResourceGraph
		constructsMapping   map[construct.ResourceId]*ExpansionSolution
		errors              error
		unsolvedConstraints []constraints.Constraint
	}

	SolveOutput struct {
		ResourceGraph *construct.ResourceGraph
		Decision      []Decision
	}

	// Decision is a struct that represents a decision made by the engine
	Decision struct {
		Level  Level
		Result DecisionResult
		Action Action
		Cause  Cause
	}
	Action string
	Cause  struct {
		Expansion           *graph.Edge[construct.Resource]
		Configuration       *graph.Edge[construct.Resource]
		OperationalResource construct.Resource
		Constraint          *constraints.Constraint
	}
	DecisionResult struct {
		Resource construct.Resource
		Edge     graph.Edge[construct.Resource]
	}
	Level string
)

func NewEngine(providers map[string]provider.Provider, kb knowledgebase.EdgeKB, constructs []construct.Construct) *Engine {
	engine := &Engine{
		Providers:              providers,
		KnowledgeBase:          kb,
		Constructs:             constructs,
		ClassificationDocument: classification.BaseClassificationDocument,
	}
	_ = engine.LoadGuardrails([]byte(""))
	engine.ResourceTemplates = make(map[construct.ResourceId]*knowledgebase.ResourceTemplate)
	for _, p := range providers {
		for id, template := range p.GetOperationalTemplates() {
			engine.ResourceTemplates[id] = template
		}
	}
	engine.EdgeTemplates = make(map[string]*knowledgebase.EdgeTemplate)
	for _, p := range providers {
		for tempKey, template := range p.GetEdgeTemplates() {
			if _, ok := engine.EdgeTemplates[tempKey]; ok {
				zap.S().Errorf("got duplicate edge template for %s", tempKey)
			}
			engine.EdgeTemplates[tempKey] = template
			srcRes, err := engine.Providers[template.Source.Provider].CreateConstructFromId(template.Source, engine.Context.InitialState)
			if err != nil {
				zap.S().Errorf("got error when creating resource from id %s, err: %s", template.Source, err.Error())
				continue
			}
			dstRes, err := engine.Providers[template.Destination.Provider].CreateConstructFromId(template.Destination, engine.Context.InitialState)
			if err != nil {
				zap.S().Errorf("got error when creating resource from id %s, err: %s", template.Destination, err.Error())
				continue
			}
			edge := knowledgebase.Edge{
				Source:      reflect.TypeOf(srcRes),
				Destination: reflect.TypeOf(dstRes),
			}
			engine.KnowledgeBase.EdgeMap[edge] = knowledgebase.EdgeDetails{
				DirectEdgeOnly:          template.DirectEdgeOnly,
				DeploymentOrderReversed: template.DeploymentOrderReversed,
				DeletetionDependent:     template.DeletetionDependent,
				Reuse:                   template.Reuse,
				Configure:               engine.KnowledgeBase.EdgeMap[edge].Configure,
			}

			if engine.KnowledgeBase.EdgesByType[reflect.TypeOf(srcRes)] == nil {
				engine.KnowledgeBase.EdgesByType[reflect.TypeOf(srcRes)] = &knowledgebase.ResourceEdges{}
			}
			engine.KnowledgeBase.EdgesByType[reflect.TypeOf(srcRes)].Outgoing = append(engine.KnowledgeBase.EdgesByType[reflect.TypeOf(srcRes)].Outgoing, edge)
			if engine.KnowledgeBase.EdgesByType[reflect.TypeOf(dstRes)] == nil {
				engine.KnowledgeBase.EdgesByType[reflect.TypeOf(dstRes)] = &knowledgebase.ResourceEdges{}
			}
			engine.KnowledgeBase.EdgesByType[reflect.TypeOf(dstRes)].Incoming = append(engine.KnowledgeBase.EdgesByType[reflect.TypeOf(dstRes)].Incoming, edge)
		}
	}
	return engine
}

func (e *Engine) LoadClassifications(classificationPath string, fs embed.FS) error {
	var err error
	e.ClassificationDocument, err = classification.ReadClassificationDoc(classificationPath, fs)
	return err
}

func (e *Engine) ContextFromInput(input input.Input) (err error) {
	e.Context = EngineContext{
		Input:       input,
		Constraints: input.Constraints.ByScope(),
	}
	e.Context.InitialState, err = input.Load(e.Providers)
	if err != nil {
		return
	}
	e.Context.WorkingState = e.Context.InitialState.Clone()
	return nil
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
func (e *Engine) Run() (*construct.ResourceGraph, error) {

	//Validate all resources used in constraints are allowed
	err := e.checkIfConstraintsAreAllowed()
	if err != nil {
		return nil, err
	}

	// First we look at all application constraints to see what is going to be added and removed from the construct graph
	for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
		err := e.ApplyApplicationConstraint(constraint.(*constraints.ApplicationConstraint))
		if err != nil {
			return nil, err
		}
	}

	// These edge constraints are at a construct level
	var joinedErr error
	for _, constraint := range e.Context.Constraints[constraints.EdgeConstraintScope] {
		err := e.ApplyEdgeConstraint(constraint.(*constraints.EdgeConstraint))
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
	}
	if joinedErr != nil {
		return nil, joinedErr
	}

	zap.S().Debug("Engine Expanding constructs")
	err = e.ExpandConstructs()
	if err != nil {
		return nil, err
	}
	zap.S().Debug("Engine done Expanding constructs")
	contextsToSolve, err := e.GenerateCombinations()
	if err != nil {
		return nil, err
	}
	for _, context := range contextsToSolve {
		solution, err := e.SolveGraph(context)
		if err != nil {
			zap.S().Debugf("got error when solving graph, with context %s, err: %s", context, err.Error())
			continue
		}
		if e.Context.Solution == nil {
			e.Context.Solution = solution
		}
		break
	}
	if e.Context.Solution == nil {
		var closestSolvedContext *SolveContext
		for _, context := range contextsToSolve {
			if closestSolvedContext == nil {
				closestSolvedContext = context
			}
			if len(context.unsolvedConstraints) < len(closestSolvedContext.unsolvedConstraints) {
				closestSolvedContext = context
			}
		}
		errorString := "no valid graphs found"
		if closestSolvedContext.unsolvedConstraints != nil {
			errorString = fmt.Sprintf("%s, was unable to satisfy the following constraints: %s", errorString, closestSolvedContext.unsolvedConstraints)
		}
		if closestSolvedContext.errors != nil {
			errorString = fmt.Sprintf("%s.\ngot the following errors when solving the graph %s", errorString, closestSolvedContext.errors.Error())
		}

		return nil, fmt.Errorf(errorString)
	}

	return e.Context.Solution, nil
}

func (e *Engine) GenerateCombinations() ([]*SolveContext, error) {
	var joinedErr error
	toSolve := []*SolveContext{}
	baseGraph := construct.NewResourceGraph()
	for _, res := range e.Context.WorkingState.ListConstructs() {
		if res.Id().Provider != construct.AbstractConstructProvider {
			resource, ok := res.(construct.Resource)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("construct %s is not a resource", res.Id()))
				continue
			}
			baseGraph.AddResource(resource)
		}
	}
	for _, dep := range e.Context.WorkingState.ListDependencies() {
		if dep.Source.Id().Provider != construct.AbstractConstructProvider && dep.Destination.Id().Provider != construct.AbstractConstructProvider {
			baseGraph.AddDependencyWithData(dep.Source.(construct.Resource), dep.Destination.(construct.Resource), dep.Properties.Data)
		}
	}
	if len(e.Context.constructExpansionSolutions) == 0 {
		return []*SolveContext{{ResourceGraph: baseGraph}}, nil
	}
	var combinations []map[construct.ResourceId]*ExpansionSolution
	for resId, sol := range e.Context.constructExpansionSolutions {
		if len(combinations) == 0 {
			for _, s := range sol {
				combinations = append(combinations, map[construct.ResourceId]*ExpansionSolution{resId: s})
			}
		} else {
			var newCombinations []map[construct.ResourceId]*ExpansionSolution
			for _, comb := range combinations {
				for _, s := range sol {
					newComb := make(map[construct.ResourceId]*ExpansionSolution)
					for k, v := range comb {
						newComb[k] = v
					}
					newComb[resId] = s
					newCombinations = append(newCombinations, newComb)
				}
			}
			combinations = newCombinations
		}
	}
	for _, comb := range combinations {
		rg := baseGraph.Clone()
		mappedRes := map[construct.ResourceId][]construct.Resource{}
		// we will clone resources otherwise we will have side effects as we solve context by context due to pointing at the same resource
		clonedRes := map[construct.ResourceId]construct.Resource{}
		for resId, sol := range comb {
			for _, res := range sol.Graph.ListResources() {
				copiedRes := cloneResource(res)
				clonedRes[res.Id()] = copiedRes
				rg.AddResource(copiedRes)
			}
			for _, edge := range sol.Graph.ListDependencies() {
				src := clonedRes[edge.Source.Id()]
				dst := clonedRes[edge.Destination.Id()]
				rg.AddDependencyWithData(src, dst, edge.Properties.Data)
			}
			mappedRes[resId] = sol.DirectlyMappedResources
		}

		for _, dep := range e.Context.WorkingState.ListDependencies() {
			if dep.Source.Id().Provider != construct.AbstractConstructProvider && dep.Destination.Id().Provider != construct.AbstractConstructProvider {
				continue
			}

			srcNodes := []construct.Resource{}
			dstNodes := []construct.Resource{}
			if dep.Source.Id().Provider == construct.AbstractConstructProvider {
				srcResources, ok := mappedRes[dep.Source.Id()]
				if !ok {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resources for construct %s", dep.Source.Id()))
					continue
				}
				for _, res := range srcResources {
					// we will clone resources otherwise we will have side effects as we solve context by context due to pointing at the same resource
					srcNodes = append(srcNodes, cloneResource(res))
				}
			} else {
				srcClone := cloneResource(dep.Source.(construct.Resource))
				srcNodes = append(srcNodes, srcClone)
			}

			if dep.Destination.Id().Provider == construct.AbstractConstructProvider {
				dstResources, ok := mappedRes[dep.Destination.Id()]
				if !ok {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resources for construct %s", dep.Destination.Id()))
					continue
				}
				for _, res := range dstResources {
					// we will clone resources otherwise we will have side effects as we solve context by context due to pointing at the same resource
					dstNodes = append(dstNodes, cloneResource(res))
				}
			} else {
				dstClone := cloneResource(dep.Destination.(construct.Resource))
				dstNodes = append(dstNodes, dstClone)
			}

			for _, srcNode := range srcNodes {
				for _, dstNode := range dstNodes {
					rg.AddDependencyWithData(srcNode, dstNode, dep.Properties.Data)
				}
			}
		}
		toSolve = append(toSolve, &SolveContext{
			ResourceGraph:     rg,
			constructsMapping: comb,
		})
	}
	return toSolve, joinedErr
}

func (e *Engine) SolveGraph(context *SolveContext) (*construct.ResourceGraph, error) {
	NUM_LOOPS := 10
	graph := context.ResourceGraph
	errorMap := make(map[int][]error)
	var configuredEdges map[construct.ResourceId]map[construct.ResourceId]bool
	for i := 0; i < NUM_LOOPS; i++ {
		for _, rc := range e.Context.Constraints[constraints.ResourceConstraintScope] {
			err := e.ApplyResourceConstraint(graph, rc.(*constraints.ResourceConstraint))
			if err != nil {
				errorMap[i] = append(errorMap[i], err)
			}
		}
		err := e.expandEdges(graph)
		if err != nil {
			errorMap[i] = append(errorMap[i], err)
		} else {
			configuredEdges, err = e.configureEdges(graph)
			if err != nil {
				errorMap[i] = append(errorMap[i], err)
			}
		}

		zap.S().Debug("Engine done configuring edges")
		operationalResources, err := e.MakeResourcesOperational(graph)
		if err != nil {
			errorMap[i] = append(errorMap[i], err)
		}
		zap.S().Debug("Validating constraints")
		unsatisfiedConstraints := e.ValidateConstraints(context)

		var joinedErr error
		for _, error := range errorMap[i] {
			joinedErr = errors.Join(joinedErr, error)
		}
		context.errors = joinedErr
		if len(unsatisfiedConstraints) > 0 && i == NUM_LOOPS-1 {
			constraintsString := ""
			for _, constraint := range unsatisfiedConstraints {
				constraintsString += fmt.Sprintf("%s\n", constraint)
			}
			zap.S().Debugf("unsatisfied constraints: %s", constraintsString)
			context.unsolvedConstraints = unsatisfiedConstraints
			return graph, fmt.Errorf("unsatisfied constraints: %s", constraintsString)
		} else {
			// check to make sure that every resource is operational
			notOperationalList := make([]string, 0)
			for _, res := range graph.ListResources() {
				if !operationalResources[res.Id()] {
					notOperationalList = append(notOperationalList, res.Id().String())
				}
			}
			if len(notOperationalList) > 0 {
				errorMap[i] = append(errorMap[i], fmt.Errorf("the following resources are not operational: %s", strings.Join(notOperationalList, ", ")))
			}
			// check to make sure that each edge is configured
			notConfiguredList := make([]string, 0)
			for _, dep := range graph.ListDependencies() {
				if !configuredEdges[dep.Source.Id()][dep.Destination.Id()] {
					notConfiguredList = append(notConfiguredList, fmt.Sprintf("%s -> %s", dep.Source.Id(), dep.Destination.Id()))
				}
			}
			if len(notConfiguredList) > 0 {
				errorMap[i] = append(errorMap[i], fmt.Errorf("the following edges are not configured: %s", strings.Join(notConfiguredList, ", ")))
			}
			if len(errorMap[i]) == 0 {
				break
			}
			var joinedErr error
			for _, error := range errorMap[i] {
				joinedErr = errors.Join(joinedErr, error)
			}
			context.errors = joinedErr
			if i == NUM_LOOPS-1 {
				return nil, fmt.Errorf("found the following errors during graph solving: %s", context.errors.Error())
			} else {
				zap.S().Debugf("got errors: %s", joinedErr.Error())
			}
		}
	}
	zap.S().Debug("Validated constraints")
	return graph, nil
}

// ApplyApplicationConstraint applies an application constraint to the either the engines working state construct graph
//
// Currently ApplicationConstraints can only be applied if the representing nodes are klotho constructs and not provider level resources
func (e *Engine) ApplyApplicationConstraint(constraint *constraints.ApplicationConstraint) error {
	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		construct, err := e.CreateConstructFromId(constraint.Node)
		if err != nil {
			return err
		}
		e.Context.WorkingState.AddConstruct(construct)

	case constraints.RemoveConstraintOperator:
		resource := e.Context.WorkingState.GetConstruct(constraint.Node)
		if resource == nil {
			return fmt.Errorf("construct, %s, does not exist", constraint.Node)
		}
		if !e.deleteConstruct(resource, true, true) {
			return fmt.Errorf("cannot remove construct %s, failed", constraint.Node)
		}
		return nil

	case constraints.ReplaceConstraintOperator:
		c := e.Context.WorkingState.GetConstruct(constraint.Node)
		if c == nil {
			return fmt.Errorf("construct, %s, does not exist", c.Id())
		}
		replacement, err := e.CreateConstructFromId(constraint.ReplacementNode)
		if err != nil {
			return err
		}
		upstream := e.Context.WorkingState.GetUpstreamConstructs(c)
		downstream := e.Context.WorkingState.GetDownstreamConstructs(c)
		err = e.Context.WorkingState.RemoveConstructAndEdges(c)
		if err != nil {
			return err
		}
		var reconnectToUpstream []construct.BaseConstruct
		for _, up := range upstream {
			deleted := e.deleteConstruct(up, false, false)
			if deleted {
				reconnectToUpstream = append(reconnectToUpstream, e.Context.WorkingState.GetUpstreamConstructs(up)...)
			} else {
				reconnectToUpstream = append(reconnectToUpstream, up)
			}
		}
		var reconnectToDownstream []construct.BaseConstruct
		for _, down := range downstream {
			deleted := e.deleteConstruct(down, false, false)
			if deleted {
				reconnectToDownstream = append(reconnectToDownstream, e.Context.WorkingState.GetDownstreamConstructs(down)...)
			} else {
				reconnectToDownstream = append(reconnectToDownstream, down)
			}
		}
		e.Context.WorkingState.AddConstruct(replacement)
		for _, up := range reconnectToUpstream {
			if e.Context.WorkingState.GetConstruct(up.Id()) == nil {
				continue
			}
			e.Context.WorkingState.AddDependency(up.Id(), replacement.Id())
		}
		for _, down := range reconnectToDownstream {
			if e.Context.WorkingState.GetConstruct(down.Id()) == nil {
				continue
			}
			e.Context.WorkingState.AddDependency(replacement.Id(), down.Id())
		}

		return nil
	}
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
	if e.Context.WorkingState.GetConstruct(constraint.Target.Source) == nil {
		node, err := e.CreateConstructFromId(constraint.Target.Source)
		if err != nil {
			return err
		}
		e.Context.WorkingState.AddConstruct(node)
	}
	if e.Context.WorkingState.GetConstruct(constraint.Target.Target) == nil {
		node, err := e.CreateConstructFromId(constraint.Target.Target)
		if err != nil {
			return err
		}
		e.Context.WorkingState.AddConstruct(node)
	}
	switch constraint.Operator {
	case constraints.MustExistConstraintOperator:
		e.Context.WorkingState.AddDependencyWithData(constraint.Target.Source, constraint.Target.Target, knowledgebase.EdgeData{Attributes: constraint.Attributes})
	case constraints.MustNotExistConstraintOperator:

		paths, err := e.Context.WorkingState.AllPaths(constraint.Target.Source, constraint.Target.Target)
		if err != nil {
			return err
		}

		// first we will remove all dependencies that make up the paths from the constraints source to target
		for _, path := range paths {
			var prevRes construct.BaseConstruct
			for _, res := range path {
				if prevRes != nil {
					err := e.Context.WorkingState.RemoveDependency(prevRes.Id(), res.Id())
					if err != nil {
						return err
					}
				}
				prevRes = res
			}
		}

		// Next we will try to delete any node in those paths in case they no longer are required for the architecture
		// We will pass the explicit field as false so that explicitly added resources do not get deleted
		for _, path := range paths {
			for _, res := range path {
				resource := e.Context.WorkingState.GetConstruct(res.Id())
				if resource != nil {
					e.deleteConstruct(resource, false, false)
				}
			}
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
	return nil
}

// handleEdgeConstainConstraint applies an edge constraint to the either the engines working state construct graph or end state resource graph
func (e *Engine) handleEdgeConstainConstraint(constraint *constraints.EdgeConstraint) error {
	resource, err := e.CreateResourceFromId(constraint.Node)
	if err != nil {
		return err
	}
	var data knowledgebase.EdgeData
	dep := e.Context.WorkingState.GetDependency(constraint.Target.Source, constraint.Target.Target)
	if dep == nil {
		switch constraint.Operator {
		case constraints.MustContainConstraintOperator:
			data = knowledgebase.EdgeData{
				Constraint: knowledgebase.EdgeConstraint{
					NodeMustExist: []construct.Resource{resource},
				},
			}
		case constraints.MustNotContainConstraintOperator:
			data = knowledgebase.EdgeData{
				Constraint: knowledgebase.EdgeConstraint{
					NodeMustNotExist: []construct.Resource{resource},
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
		switch constraint.Operator {
		case constraints.MustContainConstraintOperator:
			data.Constraint.NodeMustExist = append(data.Constraint.NodeMustExist, resource)
		case constraints.MustNotContainConstraintOperator:
			data.Constraint.NodeMustNotExist = append(data.Constraint.NodeMustNotExist, resource)
		}
	}
	for key, attribute := range constraint.Attributes {
		if v, ok := data.Attributes[key]; ok {
			if v != attribute {
				return fmt.Errorf("attribute %s has conflicting values. %s != %s", key, v, attribute)
			}
		}
		data.Attributes[key] = attribute
	}
	e.Context.WorkingState.AddDependencyWithData(constraint.Target.Source, constraint.Target.Target, data)
	return nil
}

func (e *Engine) ApplyResourceConstraint(graph *construct.ResourceGraph, constraint *constraints.ResourceConstraint) error {
	resource := graph.GetResource(constraint.Target)
	if resource == nil {
		return fmt.Errorf("resource %s does not exist", constraint.Target)
	}
	err := ConfigureField(resource, constraint.Property, constraint.Value, true, graph)
	if err != nil {
		return err
	}
	return nil
}

// ValidateConstraints validates all constraints against the end state resource graph
// It returns any constraints which were not satisfied by resource graphs current state
func (e *Engine) ValidateConstraints(context *SolveContext) []constraints.Constraint {
	var unsatisfied []constraints.Constraint
	for _, contextConstraints := range e.Context.Constraints {
		for _, constraint := range contextConstraints {
			mappedRes := map[construct.ResourceId][]construct.Resource{}
			for resId, sol := range context.constructsMapping {
				mappedRes[resId] = sol.DirectlyMappedResources
			}
			if !constraint.IsSatisfied(context.ResourceGraph, e.KnowledgeBase, mappedRes, e.ClassificationDocument) {
				unsatisfied = append(unsatisfied, constraint)
			}
		}

	}
	return unsatisfied
}

func (e *Engine) CreateConstructFromId(id construct.ResourceId) (construct.BaseConstruct, error) {
	provider, ok := e.Providers[id.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider %s", id.Provider)
	}
	return provider.CreateConstructFromId(id, e.Context.WorkingState)
}

func (e *Engine) CreateResourceFromId(id construct.ResourceId) (construct.Resource, error) {
	c, err := e.CreateConstructFromId(id)
	if err != nil {
		return nil, err
	}
	if r, ok := c.(construct.Resource); ok {
		return r, nil
	}
	return nil, fmt.Errorf("construct %s is not a resource (was %T)", id, c)
}

func (e *Engine) checkIfConstraintsAreAllowed() error {
	joinedErr := error(nil)
	for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
		switch c := constraint.(type) {
		case *constraints.ApplicationConstraint:
			if c.Node.Provider != construct.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Node) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in application constraint %s", c.Node, c))
			}
			if c.ReplacementNode.Provider != construct.AbstractConstructProvider && (c.ReplacementNode != construct.ResourceId{}) && !e.Guardrails.IsResourceAllowed(c.ReplacementNode) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in application constraint %s", c.ReplacementNode, c))
			}

		case *constraints.EdgeConstraint:
			if c.Node.Provider != construct.AbstractConstructProvider && (c.Node != construct.ResourceId{}) && !e.Guardrails.IsResourceAllowed(c.Node) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in edge constraint %s", c.Node, c))
			}
			if c.Target.Source.Provider != construct.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Target.Source) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in edge constraint %s", c.Target.Source, c))
			}
			if c.Target.Target.Provider != construct.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Target.Target) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in edge constraint %s", c.Target.Target, c))
			}
		case *constraints.ResourceConstraint:
			if c.Target.Provider != construct.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Target) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in resource constraint %s", c.Target, c))
			}
		}
	}
	return joinedErr
}
