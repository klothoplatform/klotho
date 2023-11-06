package operational_eval

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type edgeVertex struct {
	Edge construct.SimpleEdge

	Rules []knowledgebase.OperationalRule
}

func (ev edgeVertex) Key() Key {
	return Key{Edge: ev.Edge}
}

func (ev *edgeVertex) Dependencies(ctx solution_context.SolutionContext) (vertexDependencies, error) {
	cfgCtx := solution_context.DynamicCtx(ctx)
	propCtx := &fauxConfigContext{
		inner: cfgCtx,
		deps:  newDeps(),
	}

	data := knowledgebase.DynamicValueData{Edge: &construct.Edge{Source: ev.Edge.Source, Target: ev.Edge.Target}}

	var errs error
	for _, rule := range ev.Rules {
		err := errors.Join(errs, propCtx.ExecuteOpRule(data, rule))
		if errs != nil {
			errs = errors.Join(errs, err)
		}
	}
	if errs != nil {
		return propCtx.deps, fmt.Errorf(
			"could not execute dependencies for edge %s -> %s: %w",
			ev.Edge.Source, ev.Edge.Target, errs,
		)
	}

	return propCtx.deps, nil
}

func (ev *edgeVertex) UpdateFrom(other Vertex) {
	if ev == other {
		return
	}
	otherEdge, ok := other.(*edgeVertex)
	if !ok {
		panic(fmt.Sprintf("cannot merge edge with non-edge vertex: %T", other))
	}
	if ev.Edge != otherEdge.Edge {
		panic(fmt.Sprintf("cannot merge edges with different refs: %s != %s", ev.Edge, otherEdge.Edge))
	}
	ev.Rules = append(ev.Rules, otherEdge.Rules...)
}

func (ev *edgeVertex) Evaluate(eval *Evaluator) error {
	zap.S().With("op", "eval").Debugf("Evaluating %s", ev.Edge)
	edge := &construct.Edge{Source: ev.Edge.Source, Target: ev.Edge.Target}

	opCtx := operational_rule.OperationalRuleContext{
		Solution: eval.Solution.With("edge", edge),
		Data: knowledgebase.DynamicValueData{
			Edge: edge,
		},
	}

	var errs error
	for _, rule := range ev.Rules {
		err := opCtx.HandleOperationalRule(rule)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"could not apply edge %s -> %s operational rule: %w",
				ev.Edge.Source, ev.Edge.Target, err,
			))
		}
	}
	return errs
}
