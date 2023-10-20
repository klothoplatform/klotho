package property_eval

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"go.uber.org/zap"
)

func (eval *PropertyEval) Evaluate() error {
	defer eval.writeGraph("property_deps")
	for {
		size, err := eval.unevaluated.Order()
		if err != nil {
			return err
		}
		if size == 0 {
			return nil
		}
		ready, err := eval.popReady()
		if err != nil {
			return err
		}
		if len(ready) == 0 {
			return fmt.Errorf("possible circular dependency detected in properties graph: %d remaining", size)
		}

		evaluated := make([]EvaluationKey, len(ready))
		for i, v := range ready {
			evaluated[i] = v.Key()
		}
		eval.evaluatedOrder = append(eval.evaluatedOrder, evaluated)

		var errs error
		for i, v := range ready {
			errs = errors.Join(errs, v.Evaluate(eval))
			evaluated[i] = v.Key()
		}
		if errs != nil {
			return errs
		}

		if err := eval.RecalculateUnevaluated(); err != nil {
			return err
		}
	}
}

func (eval *PropertyEval) popReady() ([]EvaluationVertex, error) {
	adj, err := eval.unevaluated.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	var readyKeys []EvaluationKey

	for v, deps := range adj {
		if len(deps) == 0 {
			readyKeys = append(readyKeys, v)
		}
	}

	ready := make([]EvaluationVertex, 0, len(readyKeys))
	graphOps := make([]EvaluationVertex, 0, len(readyKeys))
	var errs error
	for _, key := range readyKeys {
		v, err := eval.unevaluated.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if state, ok := v.(*graphStateVertex); ok {
			stateReady, err := state.Test(eval.Solution.DataflowGraph())
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			if stateReady {
				ready = append(ready, state)
			} else {
				graphOps = append(graphOps, state)
			}
		} else {
			ready = append(ready, v)
		}
	}
	if errs != nil {
		return nil, errs
	}

	if len(ready) == 0 {
		ready = graphOps
		zap.S().With("op", "dequeue").Debugf("Only graph ops left, dequeued %d", len(ready))
	} else {
		zap.S().With("op", "dequeue").Debugf("Dequeued %d, graph ops left: %d", len(ready), len(graphOps))
	}

	for _, v := range ready {
		errs = errors.Join(errs, graph_addons.RemoveVertexAndEdges(eval.unevaluated, v.Key()))
		zap.S().With("op", "dequeue").Debugf(" - %s", v.Key())
	}

	return ready, errs
}

// RecalculateUnevaluated is used to recalculate the dependencies of all the unevaluated vertices in case
// some parts have "opened up" due to the evaluation of other vertices via template `{{ if }}` conditions or
// chained dependencies (eg `{{ fieldValue "X" (fieldValue "SomeRef" .Self) }}`, the dependency of X won't be
// able to be resolved until SomeRef is evaluated).
// There is likely a way to determine which vertices need to be recalculated, but the runtime impact of just
// recalculating them all isn't large at the size of graphs we're currently running with.
// Running on a medium sized input, this accounted for 0.18s of the total 0.69s, or ~26% of the runtime.
func (eval *PropertyEval) RecalculateUnevaluated() error {
	zap.S().Debug("Recalculating unevaluated graph for updated dependencies")
	topo, err := graph.TopologicalSort(eval.unevaluated)
	if err != nil {
		return err
	}

	dyn := solution_context.DynamicCtx(eval.Solution)

	vs := make(verticesAndDeps)
	var errs error
	for _, key := range topo {
		vertex, err := eval.unevaluated.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		errs = errors.Join(errs, vs.AddDependencies(dyn, vertex))
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(vs)
}
