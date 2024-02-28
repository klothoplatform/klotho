package operational_eval

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

type edgeVertex struct {
	Edge construct.SimpleEdge

	Rules []knowledgebase.OperationalRule
}

func (ev edgeVertex) Key() Key {
	return Key{Edge: ev.Edge}
}

func (ev *edgeVertex) Dependencies(eval *Evaluator, propCtx dependencyCapturer) error {
	data := knowledgebase.DynamicValueData{
		Edge: &construct.Edge{Source: ev.Edge.Source, Target: ev.Edge.Target},
	}

	var errs error
	for _, rule := range ev.Rules {
		errs = errors.Join(errs, propCtx.ExecuteOpRule(data, rule))
	}
	if errs != nil {
		return fmt.Errorf(
			"could not execute dependencies for edge %s -> %s: %w",
			ev.Edge.Source, ev.Edge.Target, errs,
		)
	}

	// NOTE: begin hack - this is to help resolve the api_deployment#Triggers case
	// The dependency graph doesn't currently handle this because:
	// 1. Expand(api -> lambda) depends on Subnets to determine if it's reusable
	// 2. Subnets depends on graphstate hasDownstream(vpc, lambda)
	// 3. deploy -> api depends on graphstate allDownstream(integration, api)
	//   to add the deploy -> integration edges
	// 4. deploy -> integration sets #Triggers
	//
	//
	pred, err := eval.graph.PredecessorMap()
	if err != nil {
		return err
	}
	propChanges := propCtx.GetChanges()
	for src := range propChanges.edges {
		isEvaluated, err := eval.isEvaluated(src)
		if err == nil && isEvaluated && len(pred[src]) == 0 {
			// this is okay, since it has no dependencies then changing it during evaluation
			// won't impact anything. Remove the dependency, since we'll handle it in this vertex's
			// Evaluate
			delete(propChanges.edges, src)
		}
	}

	return nil
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

		if len(rule.Steps) > 0 {
			err := opCtx.HandleOperationalRule(rule)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"could not apply edge %s operational rule: %w",
					ev.Edge, err,
				))
				continue
			}
		}

		// the configurations that are returned can be executed out of band from the property vertex
		// since the property vertex has already been evaluated. This is a hack to get around improper dep ordering
		configuration, err := addConfigurationRuleToPropertyVertex(
			knowledgebase.OperationalRule{
				If:                 rule.If,
				ConfigurationRules: configRules,
			},
			ev, cfgCtx, opCtx.Data, eval)

		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"could not apply edge %s configuration rule: %w",
				ev.Edge, err,
			))
		}

		rule.Steps = nil
		for res, configRules := range configuration {
			opCtx.Data.Resource = res
			rule.ConfigurationRules = configRules
			err := opCtx.HandleOperationalRule(rule)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"could not apply edge %s (res: %s) operational rule: %w",
					ev.Edge, res, err,
				))
				continue
			}
		}
	}

	if errs != nil {
		return errs
	}

	src, err := eval.Solution.DataflowGraph().Vertex(ev.Edge.Source)
	if err != nil {
		return err
	}
	target, err := eval.Solution.DataflowGraph().Vertex(ev.Edge.Target)
	if err != nil {
		return err
	}

	delays, err := knowledgebase.ConsumeFromResource(
		src,
		target,
		solution_context.DynamicCtx(eval.Solution),
	)
	if err != nil {
		return err
	}
	// we add constrains for the delayed consumption here since their property has not yet been evaluated
	c := eval.Solution.Constraints()
	for _, delay := range delays {
		c.Resources = append(c.Resources, constraints.ResourceConstraint{
			Operator: constraints.AddConstraintOperator,
			Target:   delay.Resource,
			Property: delay.PropertyPath,
			Value:    delay.Value,
		})
	}
	return nil
}
