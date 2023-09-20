package engine

import (
	"embed"
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
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
		Constraints                 map[constraints.ConstraintScope][]constraints.Constraint
		InitialState                *construct.ConstructGraph
		WorkingState                *construct.ConstructGraph
		Solution                    *SolveContext
		Decisions                   []Decision
		Errors                      []EngineError
		constructExpansionSolutions map[construct.ResourceId][]*ExpansionSolution
		AppName                     string
	}

	// SolveContext is a struct that represents the context of one possible graph solution
	SolveContext struct {
		ResourceGraph       *construct.ResourceGraph
		constructsMapping   map[construct.ResourceId]*ExpansionSolution
		Decisions           []Decision
		Errors              []EngineError
		UnsolvedConstraints []constraints.Constraint
	}
)

func (e *Engine) GetTemplateForResource(resource construct.Resource) *knowledgebase.ResourceTemplate {
	return e.ResourceTemplates[construct.ResourceId{Provider: resource.Id().Provider, Type: resource.Id().Type}]
}

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

func (e *Engine) LoadContext(initialState *construct.ConstructGraph, constraints map[constraints.ConstraintScope][]constraints.Constraint, appName string) {
	e.Context = EngineContext{
		Constraints:                 constraints,
		constructExpansionSolutions: make(map[construct.ResourceId][]*ExpansionSolution),
		AppName:                     appName,
	}
	if initialState != nil {
		e.Context.InitialState = initialState
		e.Context.WorkingState = initialState.Clone()
	} else if e.Context.WorkingState == nil {
		e.Context.WorkingState = construct.NewConstructGraph()
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
	e.ExpandConstructs()
	if len(e.Context.Errors) > 0 {
		return nil, fmt.Errorf("got errors when expanding constructs: %s", e.Context.Errors)
	}
	zap.S().Debug("Engine done Expanding constructs")
	contextsToSolve := e.GenerateCombinations()
	if len(e.Context.Errors) > 0 {
		return nil, fmt.Errorf("got errors when generating combinations: %s", e.Context.Errors)
	}
	numValidGraphs := 0
	for _, context := range contextsToSolve {
		e.SolveGraph(context)
		if len(context.UnsolvedConstraints) == 0 && len(context.Errors) == 0 {
			numValidGraphs++
			if e.Context.Solution == nil {
				e.Context.Solution = context
			}
		}
	}

	if numValidGraphs == 0 {
		var closestSolvedContext *SolveContext
		for _, context := range contextsToSolve {
			if closestSolvedContext == nil {
				closestSolvedContext = context
			}
			if len(context.UnsolvedConstraints) < len(closestSolvedContext.UnsolvedConstraints) {
				closestSolvedContext = context
			}
		}
		e.Context.Solution = closestSolvedContext

		errorString := "no valid graphs found"
		if closestSolvedContext.UnsolvedConstraints != nil {
			errorString = fmt.Sprintf("%s, was unable to satisfy the following constraints: %s", errorString, closestSolvedContext.UnsolvedConstraints)
		}
		if closestSolvedContext.Errors != nil {
			solutionErrorString := ""
			for _, err := range closestSolvedContext.Errors {
				solutionErrorString = fmt.Sprintf("%s\n%s", solutionErrorString, err.Error())
			}
			errorString = fmt.Sprintf("%s.\ngot the following errors when solving the graph %s", errorString, solutionErrorString)
		}

		return nil, fmt.Errorf(errorString)
	}
	zap.S().Debugf("found %d valid graphs", numValidGraphs)

	return e.Context.Solution.ResourceGraph, nil
}

func (e *Engine) GenerateCombinations() []*SolveContext {
	toSolve := []*SolveContext{}
	baseGraph := construct.NewResourceGraph()
	for _, res := range e.Context.WorkingState.ListConstructs() {
		if res.Id().Provider != construct.AbstractConstructProvider {
			resource, ok := res.(construct.Resource)
			if !ok {
				e.Context.Errors = append(e.Context.Errors, &ConstructExpansionError{
					Construct: res,
					Cause:     fmt.Errorf("construct %s is not a resource", res.Id()),
				})
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
		return []*SolveContext{{ResourceGraph: baseGraph}}
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
		newContext := &SolveContext{
			ResourceGraph:     baseGraph.Clone(),
			constructsMapping: comb,
		}
		mappedRes := map[construct.ResourceId][]construct.Resource{}
		// we will clone resources otherwise we will have side effects as we solve context by context due to pointing at the same resource
		clonedRes := map[construct.ResourceId]construct.Resource{}
		for resId, sol := range comb {
			expandedConstruct := e.Context.WorkingState.GetConstruct(resId)
			for _, res := range sol.Graph.ListResources() {
				copiedRes := cloneResource(res)
				clonedRes[res.Id()] = copiedRes
				e.handleDecision(newContext, Decision{Level: LevelInfo, Result: &DecisionResult{Resource: copiedRes}, Action: ActionCreate, Cause: &Cause{ConstructExpansion: expandedConstruct}})
			}
			for _, edge := range sol.Graph.ListDependencies() {
				edge.Source = clonedRes[edge.Source.Id()]
				edge.Destination = clonedRes[edge.Destination.Id()]
				e.handleDecision(newContext, Decision{Level: LevelInfo, Result: &DecisionResult{Edge: &edge}, Action: ActionConnect, Cause: &Cause{ConstructExpansion: expandedConstruct}})
			}
			mappedRes[resId] = sol.DirectlyMappedResources
		}

		for _, dep := range e.Context.WorkingState.ListDependencies() {

			var constructBeingExpanded construct.BaseConstruct

			if dep.Source.Id().Provider != construct.AbstractConstructProvider && dep.Destination.Id().Provider != construct.AbstractConstructProvider {
				continue
			}

			srcNodes := []construct.Resource{}
			dstNodes := []construct.Resource{}
			if dep.Source.Id().Provider == construct.AbstractConstructProvider {
				srcResources, ok := mappedRes[dep.Source.Id()]
				if !ok {
					e.Context.Errors = append(e.Context.Errors, &ConstructExpansionError{
						Construct: dep.Source,
						Cause:     fmt.Errorf("unable to find resources for construct %s", dep.Source.Id()),
					})
					continue
				}
				for _, res := range srcResources {
					// we will clone resources otherwise we will have side effects as we solve context by context due to pointing at the same resource
					srcNodes = append(srcNodes, cloneResource(res))
				}
				constructBeingExpanded = dep.Source
			} else {
				srcClone := cloneResource(dep.Source.(construct.Resource))
				srcNodes = append(srcNodes, srcClone)
			}

			if dep.Destination.Id().Provider == construct.AbstractConstructProvider {
				dstResources, ok := mappedRes[dep.Destination.Id()]
				if !ok {
					e.Context.Errors = append(e.Context.Errors, &ConstructExpansionError{
						Construct: dep.Destination,
						Cause:     fmt.Errorf("unable to find resources for construct %s", dep.Destination.Id()),
					})
					continue
				}
				for _, res := range dstResources {
					// we will clone resources otherwise we will have side effects as we solve context by context due to pointing at the same resource
					dstNodes = append(dstNodes, cloneResource(res))
				}
				constructBeingExpanded = dep.Destination
			} else {
				dstClone := cloneResource(dep.Destination.(construct.Resource))
				dstNodes = append(dstNodes, dstClone)
			}
			for _, srcNode := range srcNodes {
				for _, dstNode := range dstNodes {
					e.handleDecision(newContext, Decision{Level: LevelInfo, Result: &DecisionResult{Edge: &graph.Edge[construct.Resource]{Source: srcNode, Destination: dstNode, Properties: dep.Properties}}, Action: ActionConnect, Cause: &Cause{ConstructExpansion: constructBeingExpanded}})
				}
			}
		}
		toSolve = append(toSolve, newContext)
	}
	return toSolve
}

func (e *Engine) SolveGraph(context *SolveContext) {
	NUM_LOOPS := 10
	graph := context.ResourceGraph
	configuredEdges := make(map[construct.ResourceId]map[construct.ResourceId]bool)
	operationalResources := make(map[construct.ResourceId]bool)

	for i := 0; i < NUM_LOOPS; i++ {
		context.Errors = []EngineError{}

		for _, r := range graph.ListResources() {
			e.configureResource(context, r)
		}

		for _, dep := range graph.ListDependencies() {
			if configuredEdges[dep.Source.Id()] == nil {
				configuredEdges[dep.Source.Id()] = make(map[construct.ResourceId]bool)
			}

			err := e.expandEdge(dep, context)
			if err != nil {
				context.Errors = append(context.Errors, err)
				continue
			}

			errs := e.configureEdge(dep, context)
			if err != nil {
				context.Errors = append(context.Errors, errs...)
				continue
			}
			configuredEdges[dep.Source.Id()][dep.Destination.Id()] = true
		}
		resources, err := context.ResourceGraph.ReverseTopologicalSort()
		if err != nil {
			context.Errors = append(context.Errors, &InternalError{
				Cause: fmt.Errorf("error sorting resources for operationalization: %w", err),
				Child: &ResourceNotOperationalError{Cause: err},
			})
		} else {
			for _, resource := range resources {
				success := e.MakeResourceOperational(context, resource)
				if success {
					operationalResources[resource.Id()] = true
				}
			}
		}

		zap.S().Debug("Validating constraints")
		unsatisfiedConstraints := e.ValidateConstraints(context)
		context.UnsolvedConstraints = unsatisfiedConstraints

		// check to make sure that every resource is operational
		for _, res := range graph.ListResources() {
			if !operationalResources[res.Id()] {
				context.Errors = append(context.Errors, &ResourceNotOperationalError{
					Resource: res,
					Cause:    fmt.Errorf("resource %s is not operational", res.Id()),
				})
			}
		}
		// check to make sure that each edge is configured
		for _, dep := range graph.ListDependencies() {
			if !configuredEdges[dep.Source.Id()][dep.Destination.Id()] {
				context.Errors = append(context.Errors, &EdgeConfigurationError{
					Edge:  dep,
					Cause: fmt.Errorf("edge %s -> %s is not configured", dep.Source.Id(), dep.Destination.Id()),
				})
			}
		}

		if len(context.Errors) == 0 && len(context.UnsolvedConstraints) == 0 {
			break
		}
	}
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
		if !e.deleteConstruct(e.Context.WorkingState, resource, true, true) {
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
			deleted := e.deleteConstruct(e.Context.WorkingState, up, false, false)
			if deleted {
				reconnectToUpstream = append(reconnectToUpstream, e.Context.WorkingState.GetUpstreamConstructs(up)...)
			} else {
				reconnectToUpstream = append(reconnectToUpstream, up)
			}
		}
		var reconnectToDownstream []construct.BaseConstruct
		for _, down := range downstream {
			deleted := e.deleteConstruct(e.Context.WorkingState, down, false, false)
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
					e.deleteConstruct(e.Context.WorkingState, resource, false, false)
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
		crefs := r.BaseConstructRefs()
		if crefs == nil {
			err = createConstructRefs(r)
			if err != nil {
				return nil, err
			}
		}
		return r, nil
	}
	return nil, fmt.Errorf("construct %s is not a resource (was %T)", id, c)
}

func createConstructRefs(r construct.Resource) error {
	v := reflect.ValueOf(r).Elem()
	f := v.FieldByName("ConstructRefs")
	if !f.IsValid() {
		return fmt.Errorf("resource %s does not have a ConstructRefs field", r.Id())
	}
	f.Set(reflect.ValueOf(make(construct.BaseConstructSet)))
	return nil
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
