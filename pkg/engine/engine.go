package engine

import (
	"context"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type (
	// Engine is a struct that represents the object which processes the resource graph and applies constraints
	Engine struct {
		Kb knowledgebase.TemplateKB
	}

	// SolveRequest is a struct that represents the context of the engine
	// The context is used to store the state of the engine
	SolveRequest struct {
		Constraints  constraints.Constraints
		InitialState construct.Graph
		GlobalTag    string
	}
)

func NewEngine(kb knowledgebase.TemplateKB) *Engine {
	return &Engine{
		Kb: kb,
	}
}

func (e *Engine) Run(ctx context.Context, req *SolveRequest) (solution.Solution, error) {
	solutionCtx := NewSolutionContext(e.Kb, req.GlobalTag, &req.Constraints)
	err := solutionCtx.LoadGraph(req.InitialState)
	if err != nil {
		return solutionCtx, err
	}
	err = ApplyConstraints(solutionCtx)
	if err != nil {
		return solutionCtx, err
	}
	err = solutionCtx.Solve()
	return solutionCtx, err
}
