package engine

import (
	"context"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"gopkg.in/yaml.v3"
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
	sol := NewSolution(ctx, e.Kb, req.GlobalTag, &req.Constraints)
	err := sol.LoadGraph(req.InitialState)
	if err != nil {
		return sol, err
	}
	err = ApplyConstraints(sol)
	if err != nil {
		return sol, err
	}
	err = sol.Solve()
	return sol, err
}

func (req SolveRequest) MarshalYAML() (interface{}, error) {
	var initState yaml.Node
	if err := initState.Encode(construct.YamlGraph{Graph: req.InitialState}); err != nil {
		return nil, fmt.Errorf("failed to marshal initial state: %w", err)
	}
	var constraints yaml.Node
	if err := constraints.Encode(req.Constraints); err != nil {
		return nil, fmt.Errorf("failed to marshal constraints: %w", err)
	}
	content := make([]*yaml.Node, 0, len(constraints.Content)+len(initState.Content))
	content = append(content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "constraints"},
		&constraints,
	)
	content = append(content, initState.Content...)
	return yaml.Node{
		Kind:    yaml.MappingNode,
		Content: content,
	}, nil
}
