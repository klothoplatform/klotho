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
		Constraints  constraints.Constraints
		InitialState construct.Graph
		Solutions    []solution_context.SolutionContext
	}
)

func NewEngine(kb knowledgebase.TemplateKB) *Engine {
	return &Engine{
		Kb: kb,
	}
}

func (e *Engine) Run(context *EngineContext) error {
	solutionCtx := NewSolutionContext(e.Kb)
	solutionCtx.constraints = &context.Constraints
	err := solutionCtx.LoadGraph(context.InitialState)
	if err != nil {
		return err
	}
	err = ApplyConstraints(solutionCtx)
	if err != nil {
		return err
	}
	err = solutionCtx.Solve()
	context.Solutions = append(context.Solutions, solutionCtx)
	return err
}
