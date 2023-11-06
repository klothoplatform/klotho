package operational_eval

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"go.uber.org/zap"
)

type (
	graphTestFunc    func(construct.Graph) (bool, error)
	graphStateRepr   string
	graphStateVertex struct {
		repr graphStateRepr
		Test graphTestFunc
	}
)

func (gv graphStateVertex) Key() Key {
	return Key{GraphState: gv.repr}
}

func (gv *graphStateVertex) Dependencies(ctx solution_context.SolutionContext) (vertexDependencies, error) {
	return vertexDependencies{}, nil
}

func (gv *graphStateVertex) UpdateFrom(other Vertex) {
	if gv.repr != other.Key().GraphState {
		panic("cannot merge graph states with different reprs")
	}
}

func (gv *graphStateVertex) Evaluate(eval *Evaluator) error {
	zap.S().With("op", "eval").Debugf("Evaluating %s", gv.repr)
	return nil
}
