package operational_eval

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type edgeVertex struct {
	Edge construct.SimpleEdge

	Rules []knowledgebase.OperationalRule
}

func (ev edgeVertex) Key() Key {
	return Key{Edge: ev.Edge}
}

func (ev *edgeVertex) Dependencies(eval *Evaluator) (graphChanges, error) {
	cfgCtx := solution_context.DynamicCtx(eval.Solution)

	changes := newChanges()
	propCtx := newDepCapture(cfgCtx, changes, ev.Key())

	data := knowledgebase.DynamicValueData{
		Edge: &construct.Edge{Source: ev.Edge.Source, Target: ev.Edge.Target},
	}

	var errs error
	for _, rule := range ev.Rules {
		err := errors.Join(errs, propCtx.ExecuteOpRule(data, rule))
		if errs != nil {
			errs = errors.Join(errs, err)
		}

		for _, config := range rule.ConfigurationRules {
			var ref construct.PropertyRef
			err = cfgCtx.ExecuteDecode(config.Config.Field, data, &ref.Property)
			if err != nil {
				continue
			}
			err := cfgCtx.ExecuteDecode(config.Resource, data, &ref.Resource)
			if err != nil {
				continue
			}
			changes.addEdge(Key{Ref: ref}, ev.Key())
		}
	}
	if errs != nil {
		return changes, fmt.Errorf(
			"could not execute dependencies for edge %s -> %s: %w",
			ev.Edge.Source, ev.Edge.Target, errs,
		)
	}

	return changes, nil
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
	edge := &construct.Edge{Source: ev.Edge.Source, Target: ev.Edge.Target}

	cfgCtx := solution_context.DynamicCtx(eval.Solution)
	opCtx := operational_rule.OperationalRuleContext{
		Solution: eval.Solution.With("edge", edge),
		Data: knowledgebase.DynamicValueData{
			Edge: edge,
		},
	}

	var errs error
	for _, rule := range ev.Rules {
		configRules := rule.ConfigurationRules
		rule.ConfigurationRules = nil

		err := opCtx.HandleOperationalRule(rule)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"could not apply edge %s -> %s operational rule: %w",
				ev.Edge.Source, ev.Edge.Target, err,
			))
		}

		for _, config := range configRules {
			var ref construct.PropertyRef
			err := cfgCtx.ExecuteDecode(config.Resource, opCtx.Data, &ref.Resource)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"could not apply edge %s -> %s configuration rule: %w",
					ev.Edge.Source, ev.Edge.Target, err,
				))
			}
			err = cfgCtx.ExecuteDecode(config.Config.Field, opCtx.Data, &ref.Property)
			if err != nil {
				continue
			}
			key := Key{Ref: ref}
			vertex, err := eval.graph.Vertex(key)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not attempt to get existing vertex for %s: %w", ref, err))
				continue
			}
			_, unevalErr := eval.unevaluated.Vertex(key)
			if errors.Is(unevalErr, graph.ErrVertexNotFound) {
				errs = errors.Join(errs, fmt.Errorf("cannot add rules to evaluated node %s for %s", ref, ev.Edge))
				continue
			} else if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not get existing unevaluated vertex for %s: %w", ref, err))
				continue
			}
			if v, ok := vertex.(*propertyVertex); ok {
				v.EdgeRules[ev.Edge] = append(v.EdgeRules[ev.Edge], knowledgebase.OperationalRule{
					If:                 rule.If,
					ConfigurationRules: []knowledgebase.ConfigurationRule{config},
				})
			} else {
				errs = errors.Join(errs, fmt.Errorf("existing vertex for %s is not a property vertex", ref))
				continue
			}
		}
	}
	return errs
}
