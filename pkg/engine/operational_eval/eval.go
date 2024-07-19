package operational_eval

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"github.com/klothoplatform/klotho/pkg/tui"
	"go.uber.org/zap"
)

func (eval *Evaluator) updateSolveProgress() error {
	prog := tui.GetProgress(eval.Solution.Context())

	size, err := eval.unevaluated.Order()
	if err != nil {
		return err
	}
	totalSize, err := eval.graph.Order()
	if err != nil {
		return err
	}
	prog.Update("Solving", totalSize-size, totalSize)
	return nil
}

func (eval *Evaluator) Evaluate() error {
	defer eval.writeGraph("property_deps")
	defer eval.writeExecOrder()

	for {
		size, err := eval.unevaluated.Order()
		if err != nil {
			return fmt.Errorf("failed to get unevaluated order: %w", err)
		}
		if size == 0 {
			return nil
		}

		// add to evaluatedOrder so that in popReady it has the correct group number
		// which is based on `len(eval.evaluatedOrder)`
		eval.evaluatedOrder = append(eval.evaluatedOrder, []Key{})

		ready, err := eval.pollReady()
		if err != nil {
			return fmt.Errorf("failed to poll ready: %w", err)
		}
		if len(ready) == 0 {
			return fmt.Errorf("possible circular dependency detected in properties graph: %d remaining", size)
		}

		log := eval.Log().Named("eval")

		groupStart := time.Now()
		var errs []error
		for _, v := range ready {
			k := v.Key()
			addErr := func(err error) {
				errs = append(errs, fmt.Errorf("failed to evaluate %s: %w", k, err))
			}
			_, err := eval.unevaluated.Vertex(k)
			switch {
			case err != nil && !errors.Is(err, graph.ErrVertexNotFound):
				addErr(err)
				continue
			case errors.Is(err, graph.ErrVertexNotFound):
				// vertex was removed by earlier ready vertex
				continue
			}
			log.Debugf("Evaluating %s", k)
			eval.evaluatedOrder[len(eval.evaluatedOrder)-1] = append(eval.evaluatedOrder[len(eval.evaluatedOrder)-1], k)
			eval.currentKey = &k
			if err := graph_addons.RemoveVertexAndEdges(eval.unevaluated, v.Key()); err != nil {
				addErr(err)
			}
			start := time.Now()
			err = v.Evaluate(eval)
			duration := time.Since(start)
			if err != nil {
				eval.errored.Add(k)
				addErr(err)
			}

			if _, props, err := eval.graph.VertexWithProperties(k); err != nil {
				log.Errorf("failed to get properties for %s: %s", k, err)
			} else {
				props.Attributes[attribDuration] = duration.String()
			}
			if err := eval.updateSolveProgress(); err != nil {
				return err
			}
		}
		log.Debugf("Completed group in %s", time.Since(groupStart))
		if len(errs) > 0 {
			return fmt.Errorf("failed to evaluate group %d: %w", len(eval.evaluatedOrder), errors.Join(errs...))
		}

		recalcStart := time.Now()
		if err := eval.RecalculateUnevaluated(); err != nil {
			return err
		}
		log.Debugf("Recalculated unevaluated in %s", time.Since(recalcStart))
	}
}

func (eval *Evaluator) printUnevaluated() {
	log := eval.Log().Named("eval.poll-deps")
	if !log.Desugar().Core().Enabled(zap.DebugLevel) {
		return
	}
	adj, err := eval.unevaluated.AdjacencyMap()
	if err != nil {
		log.Errorf("Could not get adjacency map: %s", err)
		return
	}
	keys := make([]Key, 0, len(adj))
	for k := range adj {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i].Less(keys[j])
	})
	log.Debugf("Unevaluated vertices: %d", len(keys))
	for _, k := range keys {
		srcStr := fmt.Sprintf("%s (%d)", k, len(adj[k]))
		srcV, err := eval.unevaluated.Vertex(k)
		if err != nil {
			srcStr += fmt.Sprintf(" [error: %s]", err)
		} else {
			if cond, ok := srcV.(conditionalVertex); ok {
				vReady, err := cond.Ready(eval)
				if err != nil {
					srcStr += fmt.Sprintf(" [error: %s]", err)
				} else {
					srcStr += fmt.Sprintf(" [%s]", vReady)
				}
			}
		}
		log.Debug(srcStr)
		ts := make([]Key, 0, len(adj[k]))
		for t := range adj[k] {
			ts = append(ts, t)
		}
		sort.SliceStable(ts, func(i, j int) bool {
			return ts[i].Less(ts[j])
		})
		for _, t := range ts {
			log.Debugf(" - %s", t)
		}
	}
}

func (eval *Evaluator) pollReady() ([]Vertex, error) {
	log := eval.Log().Named("eval.dequeue")
	adj, err := eval.unevaluated.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	eval.printUnevaluated()

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
			sort.SliceStable(ready, func(i, j int) bool {
				a, b := ready[i].Key(), ready[j].Key()
				return a.Less(b)
			})
			log.Debugf("Dequeued [%s]: %d", ReadyPriority(i), len(ready))
			for _, v := range ready {
				log.Debugf(" - %s", v.Key())
			}
		} else if len(prio) > 0 {
			log.Debugf("Remaining unready [%s]: %d", ReadyPriority(i), len(prio))
		}
	}

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

func (eval *Evaluator) cleanupPropertiesSubVertices(ref construct.PropertyRef, resource *construct.Resource) error {
	topo, err := graph.TopologicalSort(eval.unevaluated)
	if err != nil {
		return err
	}

	var errs error
	for _, key := range topo {
		if key.keyType() != keyTypeProperty {
			continue
		}
		if key.Ref.Resource != ref.Resource {
			continue
		}
		if strings.HasPrefix(key.Ref.Property, ref.Property) {

			path, err := resource.PropertyPath(key.Ref.Property)
			// an error would mean that the path no longer exists so we know we should remove the vertex
			if err == nil {
				// if the paths parent still exists then we know we will end up evaluating the vertex and should not remove it
				parentIndex := len(path) - 2
				if parentIndex < 0 {
					continue
				}
				if parent, ok := path[parentIndex].Get(); ok && parent != nil {
					continue
				}
			}
			errs = errors.Join(errs, graph_addons.RemoveVertexAndEdges(eval.graph, key))
			errs = errors.Join(errs, graph_addons.RemoveVertexAndEdges(eval.unevaluated, key))
		}
	}
	return errs
}
