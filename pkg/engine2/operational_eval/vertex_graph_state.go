package operational_eval

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	graphTestFunc    func(construct.Graph) (ReadyPriority, error)
	graphStateRepr   string
	graphStateVertex struct {
		repr graphStateRepr
		Test graphTestFunc
	}
)

func (gv graphStateVertex) Key() Key {
	return Key{GraphState: gv.repr}
}

func (gv *graphStateVertex) Dependencies(eval *Evaluator, propCtx dependencyCapturer) error {
	return nil
}

func (gv *graphStateVertex) UpdateFrom(other Vertex) {
	if gv.repr != other.Key().GraphState {
		panic("cannot merge graph states with different reprs")
	}
}

func (gv *graphStateVertex) Evaluate(eval *Evaluator) error {
	return nil
}

func (gv *graphStateVertex) Ready(eval *Evaluator) (ReadyPriority, error) {
	return gv.Test(eval.Solution.DataflowGraph())
}
