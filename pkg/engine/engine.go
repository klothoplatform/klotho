package engine

import (
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
		Constraints  map[constraints.ConstraintScope][]constraints.Constraint
		InitialState *core.ConstructGraph
		Dag          *core.ResourceGraph
		Decisions    []Decision
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

func (e *Engine) LoadContext(initialState *core.ConstructGraph, constraints map[constraints.ConstraintScope][]constraints.Constraint) {
	e.Context = EngineContext{
		InitialState: initialState,
		Constraints:  constraints,
		Dag:          core.NewResourceGraph(),
	}
}
