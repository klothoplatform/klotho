package operational_eval

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"github.com/klothoplatform/klotho/pkg/set"
)

func (eval *Evaluator) Evaluate() error {
	defer eval.writeGraph("property_deps")
	for {
		size, err := eval.unevaluated.Order()
		if err != nil {
			return err
		}
		if size == 0 {
			return nil
		}

		// add to evaluatedOrder so that in popReady it has the correct group number
		// which is based on `len(eval.evaluatedOrder)`
		evaluated := make(set.Set[Key])
		eval.evaluatedOrder = append(eval.evaluatedOrder, evaluated)

		ready, err := eval.pollReady()
		if err != nil {
			return err
		}
		if len(ready) == 0 {
			return fmt.Errorf("possible circular dependency detected in properties graph: %d remaining", size)
		}

		log := eval.Log().With("op", "eval")

		var errs error
		for _, v := range ready {
			k := v.Key()
			_, err := eval.unevaluated.Vertex(k)
			switch {
			case err != nil && !errors.Is(err, graph.ErrVertexNotFound):
				errs = errors.Join(errs, err)
				continue
			case errors.Is(err, graph.ErrVertexNotFound):
				// vertex was removed by earlier ready vertex
				continue
			}
			log.Debugf("Evaluating %s", k)
			evaluated.Add(k)
			eval.currentKey = &k
			err = v.Evaluate(eval)
			if err != nil {
				eval.errored.Add(k)
				errs = errors.Join(errs, fmt.Errorf("failed to evaluate %s: %w", k, err))
			}
			errs = errors.Join(errs, graph_addons.RemoveVertexAndEdges(eval.unevaluated, v.Key()))

		}
		if errs != nil {
			return fmt.Errorf("failed to evaluate group %d: %w", len(eval.evaluatedOrder), errs)
		}

		if err := eval.RecalculateUnevaluated(); err != nil {
			return err
		}
	}
}

func (eval *Evaluator) pollReady() ([]Vertex, error) {
	log := eval.Log().With("op", "dequeue")
	adj, err := eval.unevaluated.AdjacencyMap()
	if err != nil {
		return nil, err
	}

	var readyKeys []Key

	for v, deps := range adj {
		if len(deps) == 0 {
			readyKeys = append(readyKeys, v)
		}
	}

	readyPriorities := make([][]Vertex, NotReadyMax)
	var errs error
	for _, key := range readyKeys {
		v, err := eval.unevaluated.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if cond, ok := v.(conditionalVertex); ok {
			vReady, err := cond.Ready(eval)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			_, props, _ := eval.graph.VertexWithProperties(key)
			if props.Attributes != nil {
				props.Attributes[attribReady] = vReady.String()
			}
			readyPriorities[vReady] = append(readyPriorities[vReady], v)
		} else {
			readyPriorities[ReadyNow] = append(readyPriorities[ReadyNow], v)
		}
	}
	if errs != nil {
		return nil, errs
	}

	var ready []Vertex
	for i, prio := range readyPriorities {
		if len(prio) > 0 && ready == nil {
			ready = prio
			log.Debugf("Dequeued [%s]: %d", ReadyPriority(i), len(ready))
			for _, v := range ready {
				log.Debugf(" - %s", v.Key())
			}
		} else if len(prio) > 0 {
			log.Debugf("Remaining unready [%s]: %d", ReadyPriority(i), len(prio))
		}
	}

	sort.SliceStable(ready, func(i, j int) bool {
		a, b := ready[i].Key(), ready[j].Key()
		return a.Less(b)
	})

	return ready, errs
}

// RecalculateUnevaluated is used to recalculate the dependencies of all the unevaluated vertices in case
// some parts have "opened up" due to the evaluation of other vertices via template `{{ if }}` conditions or
// chained dependencies (eg `{{ fieldValue "X" (fieldValue "SomeRef" .Self) }}`, the dependency of X won't be
// able to be resolved until SomeRef is evaluated).
// There is likely a way to determine which vertices need to be recalculated, but the runtime impact of just
// recalculating them all isn't large at the size of graphs we're currently running with.
func (eval *Evaluator) RecalculateUnevaluated() error {
	topo, err := graph.TopologicalSort(eval.unevaluated)
	if err != nil {
		return err
	}

	var errs error
	for _, key := range topo {
		vertex, err := eval.unevaluated.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		changes := newChanges()
		err = changes.AddVertexAndDeps(eval, vertex)
		if err == nil {
			err = eval.enqueue(changes)
		}
		errs = errors.Join(errs, err)
	}
	return errs
}
