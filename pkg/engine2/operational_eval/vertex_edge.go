package operational_eval

import (
	"errors"
	"fmt"
	"strings"

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
		errs = errors.Join(errs, propCtx.ExecuteOpRule(data, rule))
	}
	if errs != nil {
		return changes, fmt.Errorf(
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
		return changes, err
	}
	for src := range changes.edges {
		isEvaluated, err := eval.isEvaluated(src)
		if err == nil && isEvaluated && len(pred[src]) == 0 {
			// this is okay, since it has no dependencies then changing it during evaluation
			// won't impact anything. Remove the dependency, since we'll handle it in this vertex's
			// Evaluate
			delete(changes.edges, src)
		}
	}

	if ev.Edge.Source.Type == "service" && ev.Edge.Target.Type == "pod" {
		log := eval.Log()
		log.Warnf("changes %s", ev.Edge)
		log.Warn("nodes:")
		for n := range changes.nodes {
			log.Warnf(" - %q", n)
		}
		log.Warnf("edges:")
		for src, targets := range changes.edges {
			log.Warnf("%q", src)
			for target := range targets {
				log.Warnf(" -> %q", target)
			}
		}
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
	log := eval.Log().With("op", "eval")
	edge := &construct.Edge{Source: ev.Edge.Source, Target: ev.Edge.Target}

	cfgCtx := solution_context.DynamicCtx(eval.Solution)
	opCtx := operational_rule.OperationalRuleContext{
		Solution: eval.Solution.With("edge", edge),
		Data: knowledgebase.DynamicValueData{
			Edge: edge,
		},
	}

	pred, err := eval.graph.PredecessorMap()
	if err != nil {
		return err
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

		configuration := make(map[construct.ResourceId][]knowledgebase.ConfigurationRule)
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
				var evalDeps []string
				for dep := range pred[key] {
					depEvaled, err := eval.isEvaluated(dep)
					if err != nil {
						return fmt.Errorf("could not check if %s is evaluated: %w", dep, err)
					}
					if depEvaled {
						evalDeps = append(evalDeps, `"`+dep.String()+`"`)
					}
				}
				if len(evalDeps) == 0 {
					configuration[ref.Resource] = append(configuration[ref.Resource], config)
					log.Debugf("Allowing config on %s to be evaluated due to no dependents", key)
				} else {
					errs = errors.Join(errs, fmt.Errorf(
						"cannot add rules to evaluated node %s for %s: evaluated dependents: %s",
						ref, ev.Edge, strings.Join(evalDeps, ", "),
					))
				}
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
	return errs
}
