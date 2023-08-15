package engine

import (
	"embed"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
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
		Constructs []core.Construct
		// The templates that the engine uses to make resources operational
		ResourceTemplates map[core.ResourceId]*core.ResourceTemplate
		// The templates that the engine uses to make edges operational
		EdgeTemplates map[string]*knowledgebase.EdgeTemplate
		// The context of the engine
		Context EngineContext

		Guardrails *Guardrails
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Constraints                 map[constraints.ConstraintScope][]constraints.Constraint
		InitialState                *core.ConstructGraph
		WorkingState                *core.ConstructGraph
		Solution                    *core.ResourceGraph
		Decisions                   []Decision
		constructExpansionSolutions map[core.ResourceId][]*ExpansionSolution
		AppName                     string
	}

	SolveContext struct {
		ResourceGraph       *core.ResourceGraph
		constructsMapping   map[core.ResourceId]*ExpansionSolution
		errors              error
		unsolvedConstraints []constraints.Constraint
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
	engine := &Engine{
		Providers:              providers,
		KnowledgeBase:          kb,
		Constructs:             constructs,
		ClassificationDocument: classification.BaseClassificationDocument,
	}
	_ = engine.LoadGuardrails([]byte(""))
	engine.ResourceTemplates = make(map[core.ResourceId]*core.ResourceTemplate)
	for _, p := range providers {
		for id, template := range p.GetOperationalTempaltes() {
			engine.ResourceTemplates[id] = template
		}
	}
	engine.EdgeTemplates = make(map[string]*knowledgebase.EdgeTemplate)
	for _, p := range providers {
		for tempKey, template := range p.GetEdgeTempaltes() {
			if _, ok := engine.EdgeTemplates[tempKey]; ok {
				zap.S().Errorf("got duplicate edge template for %s", tempKey)
			}
			engine.EdgeTemplates[tempKey] = template
			srcRes, err := engine.Providers[template.Source.Provider].CreateResourceFromId(template.Source, engine.Context.InitialState)
			if err != nil {
				zap.S().Errorf("got error when creating resource from id %s, err: %s", template.Source, err.Error())
				continue
			}
			dstRes, err := engine.Providers[template.Destination.Provider].CreateResourceFromId(template.Destination, engine.Context.InitialState)
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

func (e *Engine) LoadContext(initialState *core.ConstructGraph, constraints map[constraints.ConstraintScope][]constraints.Constraint, appName string) {
	e.Context = EngineContext{
		InitialState:                initialState,
		Constraints:                 constraints,
		WorkingState:                initialState.Clone(),
		constructExpansionSolutions: make(map[core.ResourceId][]*ExpansionSolution),
		AppName:                     appName,
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
	numValidGraphs := 0
	for _, context := range contextsToSolve {
		solution, err := e.SolveGraph(context)
		if err != nil {
			zap.S().Debugf("got error when solving graph, with context %s, err: %s", context, err.Error())
			continue
		}
		if e.Context.Solution == nil {
			e.Context.Solution = solution
		}
		numValidGraphs++
	}
	if numValidGraphs == 0 {
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
	zap.S().Debugf("found %d valid graphs", numValidGraphs)

	return e.Context.Solution, nil
}

func (e *Engine) GenerateCombinations() ([]*SolveContext, error) {
	var joinedErr error
	toSolve := []*SolveContext{}
	baseGraph := core.NewResourceGraph()
	for _, res := range e.Context.WorkingState.ListConstructs() {
		if res.Id().Provider != core.AbstractConstructProvider {
			resource, ok := res.(core.Resource)
			if !ok {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("construct %s is not a resource", res.Id()))
				continue
			}
			baseGraph.AddResource(resource)
		}
	}
	for _, dep := range e.Context.WorkingState.ListDependencies() {
		if dep.Source.Id().Provider != core.AbstractConstructProvider && dep.Destination.Id().Provider != core.AbstractConstructProvider {
			baseGraph.AddDependencyWithData(dep.Source.(core.Resource), dep.Destination.(core.Resource), dep.Properties.Data)
		}
	}
	if len(e.Context.constructExpansionSolutions) == 0 {
		return []*SolveContext{{ResourceGraph: baseGraph}}, nil
	}
	var combinations []map[core.ResourceId]*ExpansionSolution
	for resId, sol := range e.Context.constructExpansionSolutions {
		if len(combinations) == 0 {
			for _, s := range sol {
				combinations = append(combinations, map[core.ResourceId]*ExpansionSolution{resId: s})
			}
		} else {
			var newCombinations []map[core.ResourceId]*ExpansionSolution
			for _, comb := range combinations {
				for _, s := range sol {
					newComb := make(map[core.ResourceId]*ExpansionSolution)
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
		mappedRes := map[core.ResourceId][]core.Resource{}
		for resId, sol := range comb {
			for _, res := range sol.Graph.ListResources() {
				rg.AddResource(res)
			}
			for _, edge := range sol.Graph.ListDependencies() {
				rg.AddDependency(edge.Source, edge.Destination)
			}
			mappedRes[resId] = sol.DirectlyMappedResources
		}

		for _, dep := range e.Context.WorkingState.ListDependencies() {
			if dep.Source.Id().Provider != core.AbstractConstructProvider && dep.Destination.Id().Provider != core.AbstractConstructProvider {
				continue
			}

			srcNodes := []core.Resource{}
			dstNodes := []core.Resource{}
			if dep.Source.Id().Provider == core.AbstractConstructProvider {
				srcResources, ok := mappedRes[dep.Source.Id()]
				if !ok {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resources for construct %s", dep.Source.Id()))
					continue
				}
				srcNodes = append(srcNodes, srcResources...)
			} else {
				srcNodes = append(srcNodes, dep.Source.(core.Resource))
			}

			if dep.Destination.Id().Provider == core.AbstractConstructProvider {
				dstResources, ok := mappedRes[dep.Destination.Id()]
				if !ok {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("unable to find resources for construct %s", dep.Destination.Id()))
					continue
				}
				dstNodes = append(dstNodes, dstResources...)
			} else {
				dstNodes = append(dstNodes, dep.Destination.(core.Resource))
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

func (e *Engine) SolveGraph(context *SolveContext) (*core.ResourceGraph, error) {
	NUM_LOOPS := 10
	graph := context.ResourceGraph
	errorMap := make(map[int][]error)
	var configuredEdges map[core.ResourceId]map[core.ResourceId]bool
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
		resource := e.Context.WorkingState.GetConstruct(constraint.Node)
		if resource == nil {
			return fmt.Errorf("construct, %s, does not exist", constraint.Node)
		}
		if !e.deleteConstruct(resource, true, true) {
			return fmt.Errorf("cannot remove construct %s, failed", constraint.Node)
		}
		return nil

	case constraints.ReplaceConstraintOperator:
		construct := e.Context.WorkingState.GetConstruct(constraint.Node)
		if construct == nil {
			return fmt.Errorf("construct, %s, does not exist", construct.Id())
		}
		replacement, err := e.getConstructFromInputId(constraint.ReplacementNode)
		if err != nil {
			return err
		}
		decision.Construct = construct
		upstream := e.Context.WorkingState.GetUpstreamConstructs(construct)
		downstream := e.Context.WorkingState.GetDownstreamConstructs(construct)
		err = e.Context.WorkingState.RemoveConstructAndEdges(construct)
		if err != nil {
			return err
		}
		var reconnectToUpstream []core.BaseConstruct
		for _, up := range upstream {
			deleted := e.deleteConstruct(up, false, false)
			if deleted {
				reconnectToUpstream = append(reconnectToUpstream, e.Context.WorkingState.GetUpstreamConstructs(up)...)
			} else {
				reconnectToUpstream = append(reconnectToUpstream, up)
			}
		}
		var reconnectToDownstream []core.BaseConstruct
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
	if e.Context.WorkingState.GetConstruct(constraint.Target.Source) == nil {
		node, err := e.getConstructFromId(constraint.Target.Source)
		if err != nil {
			return err
		}
		e.Context.WorkingState.AddConstruct(node)
	}
	if e.Context.WorkingState.GetConstruct(constraint.Target.Target) == nil {
		node, err := e.getConstructFromId(constraint.Target.Target)
		if err != nil {
			return err
		}
		e.Context.WorkingState.AddConstruct(node)
	}
	switch constraint.Operator {
	case constraints.MustExistConstraintOperator:
		e.Context.WorkingState.AddDependencyWithData(constraint.Target.Source, constraint.Target.Target, knowledgebase.EdgeData{Attributes: constraint.Attributes})
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

	provider := e.Providers[constraint.Node.Provider]
	resource, err := provider.CreateResourceFromId(constraint.Node, e.Context.WorkingState)
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
					NodeMustExist: []core.Resource{resource},
				},
			}
		case constraints.MustNotContainConstraintOperator:
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

func (e *Engine) ApplyResourceConstraint(graph *core.ResourceGraph, constraint *constraints.ResourceConstraint) error {
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
			mappedRes := map[core.ResourceId][]core.Resource{}
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

func (e *Engine) getConstructFromId(id core.ResourceId) (core.BaseConstruct, error) {
	var construct core.BaseConstruct
	var err error
	if id.Provider == core.AbstractConstructProvider {
		construct, err = e.getConstructFromInputId(id)
		if err != nil {
			return nil, err
		}

	} else {
		provider := e.Providers[id.Provider]
		construct, err = provider.CreateResourceFromId(id, e.Context.InitialState)
		if err != nil {
			return nil, err
		}
	}
	return construct, err
}

func (e *Engine) checkIfConstraintsAreAllowed() error {
	joinedErr := error(nil)
	for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
		switch c := constraint.(type) {
		case *constraints.ApplicationConstraint:
			if c.Node.Provider != core.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Node) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in application constraint %s", c.Node, c))
			}
			if c.ReplacementNode.Provider != core.AbstractConstructProvider && (c.ReplacementNode != core.ResourceId{}) && !e.Guardrails.IsResourceAllowed(c.ReplacementNode) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in application constraint %s", c.ReplacementNode, c))
			}

		case *constraints.EdgeConstraint:
			if c.Node.Provider != core.AbstractConstructProvider && (c.Node != core.ResourceId{}) && !e.Guardrails.IsResourceAllowed(c.Node) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in edge constraint %s", c.Node, c))
			}
			if c.Target.Source.Provider != core.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Target.Source) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in edge constraint %s", c.Target.Source, c))
			}
			if c.Target.Target.Provider != core.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Target.Target) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in edge constraint %s", c.Target.Target, c))
			}
		case *constraints.ResourceConstraint:
			if c.Target.Provider != core.AbstractConstructProvider && !e.Guardrails.IsResourceAllowed(c.Target) {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s, is not allowed in resource constraint %s", c.Target, c))
			}
		}
	}
	return joinedErr
}
