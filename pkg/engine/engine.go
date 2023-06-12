package engine

import (
	"errors"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
)

type (
	// Engine is a struct that represents the object which processes the resource graph and applies constraints
	Engine struct {
		Provider      provider.Provider
		KnowledgeBase knowledgebase.EdgeKB
		Context       EngineContext
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Constraints  map[constraints.ConstraintScope][]constraints.Constraint
		InitialState *core.ConstructGraph
		Dag          *core.ResourceGraph
		Decisions    []Decision
		Resources    []core.Resource
	}

	// Decision is a struct that represents a decision made by the engine
	Decision struct {
		// The resources that was modified
		Resources []core.Resource
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

func (e *Engine) LoadContext(InitialState *core.ConstructGraph, Constraints map[constraints.ConstraintScope][]constraints.Constraint) {
	e.Context = EngineContext{
		InitialState: InitialState,
		Constraints:  Constraints,
		Dag:          core.NewResourceGraph(),
	}
}

func (e *Engine) Run(graph *core.ConstructGraph, constraints []constraints.Constraint, appName string) (*core.ResourceGraph, error) {

	dag := core.NewResourceGraph()

	// First we look at all application constraints to see what is going to be added and removed from the construct graph
	// for _, constraint := range e.Context.Constraints[constraints.ApplicationConstraintScope] {
	// 	err := constraint.Apply(e)
	// 	if err != nil {
	// 		return dag, err
	// 	}
	// }

	// for i := 1; i < 5; i++ {
	// 	zap.S().Infof("Running engine iteration %d", i)

	// 	err := e.Provider.ExpandConstructs(graph, dag)
	// 	if err != nil {
	// 		return dag, err
	// 	}

	// 	err = e.KnowledgeBase.ExpandEdges(dag, appName)
	// 	if err != nil {
	// 		return dag, err
	// 	}
	// 	validated, err := ValidateAndApply(dag, constraints)
	// 	if err != nil {
	// 		return dag, err
	// 	}
	// 	if validated {
	// 		break
	// 	}
	// }

	var configurationErr error
	for _, resource := range dag.ListResources() {
		// Here we would apply the configuration based on node constraints we got
		var configuration any
		err := dag.CallConfigure(resource, configuration)
		if err != nil {
			errors.Join(configurationErr, err)
		}
	}
	if configurationErr != nil {
		return dag, configurationErr
	}

	err := e.KnowledgeBase.ConfigureFromEdgeData(dag)
	if err != nil {
		return dag, err
	}

	return dag, nil
}

func ValidateAndApply(graph *core.ResourceGraph, constraints []constraints.Constraint) (bool, error) {
	var joinedErr error
	allSatisfied := true
	// for _, constraint := range constraints {
	// 	if !constraint.IsSatisfied(graph) {
	// 		allSatisfied = false
	// 		err := constraint.Apply(graph)
	// 		if err != nil {
	// 			return false, err
	// 		}
	// 	}
	// }
	return allSatisfied, joinedErr
}
