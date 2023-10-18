package property_eval

import (
	"errors"
	"fmt"

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
		zap.S().With("op", "dequeue").Debugf("Dequeued %d properties", len(ready))

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

	ready := make([]EvaluationVertex, len(readyKeys))
	var errs error
	for i, key := range readyKeys {
		v, err := eval.unevaluated.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		ready[i] = v
		errs = errors.Join(errs, graph_addons.RemoveVertexAndEdges(eval.unevaluated, key))
	}

	return ready, errs
}
