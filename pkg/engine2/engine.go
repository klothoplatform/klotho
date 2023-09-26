package engine2

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	// Engine is a struct that represents the object which processes the resource graph and applies constraints
	Engine struct {
		Kb knowledgebase.TemplateKB
	}

	// EngineContext is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	EngineContext struct {
		Constraints  map[constraints.ConstraintScope][]constraints.Constraint
		InitialState construct.Graph
		Solutions    []solution_context.SolutionContext
	}
)

func (e *Engine) CreateResourceFromId(id construct.ResourceId) construct.Resource {
	panic("implement me")
}

func NewEngine(kb knowledgebase.TemplateKB) *Engine {
	return &Engine{
		Kb: kb,
	}
}

func (e *Engine) Run(context EngineContext) error {
	solutionCtx := solution_context.NewSolutionContext()
	err := solutionCtx.LoadGraph(context.InitialState)
	if err != nil {
		return err
	}
	err = solutionCtx.LoadConstraints(context.Constraints)
	if err != nil {
		return err
	}
	solutionContexts, err := solutionCtx.GenerateCombinations()
	if err != nil {
		return err
	}
	for _, solutionContext := range solutionContexts {
		err := solutionContext.Solve()
		if err == nil {
			context.Solutions = append(context.Solutions, solutionContext)
		}
	}
	return nil
}
