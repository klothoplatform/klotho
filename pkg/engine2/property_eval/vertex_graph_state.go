package property_eval

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type graphStateVertex struct {
	repr string
	Test func(construct.Graph) (bool, error)
}

func (gv graphStateVertex) Key() EvaluationKey {
	return EvaluationKey{GraphState: gv.repr}
}

func (gv *graphStateVertex) Dependencies(
	cfgCtx knowledgebase.DynamicValueContext,
) (set.Set[construct.PropertyRef], graphStates, error) {
	return nil, nil, nil
}

func (gv *graphStateVertex) UpdateFrom(other EvaluationVertex) {
	if gv.repr != other.Key().GraphState {
		panic("cannot merge graph states with different reprs")
	}
}

func (gv *graphStateVertex) Evaluate(eval *PropertyEval) error {
	zap.S().With("op", "eval").Debugf("Evaluating %s", gv.repr)
	return nil
}
