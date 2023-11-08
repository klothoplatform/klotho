package operational_eval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type (
	Evaluator struct {
		Solution solution_context.SolutionContext

		// graph holds all of the property dependencies regardless of whether they've been evaluated or not
		graph Graph

		unevaluated Graph

		evaluatedOrder []set.Set[Key]
		errored        set.Set[Key]

		currentKey *Key
	}

	Key struct {
		Ref               construct.PropertyRef
		Edge              construct.SimpleEdge
		GraphState        graphStateRepr
		PathSatisfication *knowledgebase.EdgePathSatisfaction
		Internal          string
	}

	Vertex interface {
		Key() Key
		Evaluate(eval *Evaluator) error
		UpdateFrom(other Vertex)
		Dependencies(eval *Evaluator) (graphChanges, error)
	}

	conditionalVertex interface {
		Ready(*Evaluator) (ReadyPriority, error)
	}

	ReadyPriority int

	graphChanges struct {
		nodes map[Key]Vertex
		// edges is map[source]targets
		edges map[Key]set.Set[Key]
	}
)

const (
	// ReadyNow indicates the vertex is ready to be evaluated
	ReadyNow ReadyPriority = iota
	// NotReadyLow is used when it's relatively certain that the vertex will be ready, but cannot be 100% certain.
	NotReadyLow
	// NotReadyMid is for cases which don't clearly fit in [NotReadyLow] or [NotReadyHigh]
	NotReadyMid
	// NotReadyHigh is used for verticies which can almost never be 100% certain that they're correct based on the
	// current state.
	NotReadyHigh
	// NotReadyMax it reserved for when running the vertex would likely cause an error, rather than incorrect behaviour
	NotReadyMax
)

func NewEvaluator(ctx solution_context.SolutionContext) *Evaluator {
	return &Evaluator{
		Solution:    ctx,
		graph:       newGraph(nil),
		unevaluated: newGraph(nil),
		errored:     make(set.Set[Key]),
	}
}

func (key Key) String() string {
	if !key.Ref.Resource.IsZero() {
		return key.Ref.String()
	}
	if key.GraphState != "" {
		return string(key.GraphState)
	}
	if key.PathSatisfication != nil {
		args := []string{
			key.Edge.String(),
		}
		if key.PathSatisfication.Classification != "" {
			args = append(args, fmt.Sprintf("<%s>", key.PathSatisfication.Classification))
		}
		if key.PathSatisfication.AsTarget {
			args = append(args, "target")
		}
		return fmt.Sprintf("Expand(%s)", strings.Join(args, ", "))
	}
	if key.Edge != (construct.SimpleEdge{}) {
		return key.Edge.String()
	}
	if key.Internal != "" {
		return fmt.Sprintf("|%s|", key.Internal)
	}
	return "<empty>"
}

func (r ReadyPriority) String() string {
	switch r {
	case ReadyNow:
		return "ReadyNow"
	case NotReadyLow:
		return "NotReadyLow"
	case NotReadyMid:
		return "NotReadyMid"
	case NotReadyHigh:
		return "NotReadyHigh"
	case NotReadyMax:
		return "NotReadyMax"
	default:
		return fmt.Sprintf("ReadyPriority(%d)", r)
	}
}

func (eval *Evaluator) Log() *zap.SugaredLogger {
	return zap.S().With("group", len(eval.evaluatedOrder))
}

func (eval *Evaluator) isEvaluated(k Key) (bool, error) {
	_, err := eval.graph.Vertex(k)
	if errors.Is(err, graph.ErrVertexNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	_, err = eval.unevaluated.Vertex(k)
	if errors.Is(err, graph.ErrVertexNotFound) {
		return true, nil
	}
	return false, err
}

func (eval *Evaluator) addEdge(source, target Key) error {
	_, err := eval.graph.Edge(source, target)
	if err == nil {
		return nil
	}

	err = eval.graph.AddEdge(source, target, func(ep *graph.EdgeProperties) {
		ep.Attributes[attribAddedIn] = fmt.Sprintf("%d", len(eval.evaluatedOrder))
		if eval.currentKey != nil {
			ep.Attributes[attribAddedBy] = eval.currentKey.String()
		}
	})
	if err != nil {
		if errors.Is(err, graph.ErrEdgeCreatesCycle) {
			path, _ := graph.ShortestPath(eval.graph, target, source)
			pathS := make([]string, len(path))
			for i, k := range path {
				pathS[i] = `"` + k.String() + `"`
			}
			return fmt.Errorf(
				"could not add edge %q -> %q: would create cycle exiting path: %s",
				source, target, strings.Join(pathS, " -> "),
			)
		}
		// NOTE(gg): If this fails with target not in graph, then we might need to add the target in with a
		// new vertex type of "overwrite me later". It would be an odd situation though, which is why it is
		// an error for now.
		return fmt.Errorf("could not add edge %q -> %q: %w", source, target, err)
	}
	markError := func() {
		_ = eval.graph.UpdateEdge(source, target, func(ep *graph.EdgeProperties) {
			ep.Attributes[attribError] = "true"
		})
	}

	log := eval.Log().With("op", "deps")

	_, err = eval.unevaluated.Vertex(target)
	switch {
	case errors.Is(err, graph.ErrVertexNotFound):
		// the 'graph.AddEdge' succeeded, thus the target exists in the total graph
		// which means that the target vertex is done, so ignore adding the edge to the unevaluated graph
		log.Debugf("  -> %s (done)", target)

	case err != nil:
		markError()
		return fmt.Errorf("could not get unevaluated vertex %s: %w", target, err)

	default:
		log.Debugf("  -> %s", target)
		sourceEvaluated, err := eval.isEvaluated(source)
		if err != nil {
			markError()
			return fmt.Errorf("could not check if source %s is evaluated: %w", source, err)
		} else if sourceEvaluated {
			markError()
			return fmt.Errorf(
				"could not add edge %q -> %q: source is already evaluated",
				source, target)
		}
		err = eval.unevaluated.AddEdge(source, target)
		if err != nil {
			markError()
			return fmt.Errorf("could not add unevaluated edge %q -> %q: %w", source, target, err)
		}
	}
	return nil
}

func (eval *Evaluator) enqueue(changes graphChanges) error {
	log := eval.Log().With("op", "enqueue")

	var errs error
	for key, v := range changes.nodes {
		_, err := eval.graph.Vertex(key)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			err := eval.graph.AddVertex(v, func(vp *graph.VertexProperties) {
				vp.Attributes[attribAddedIn] = fmt.Sprintf("%d", len(eval.evaluatedOrder))
				if eval.currentKey != nil {
					vp.Attributes[attribAddedBy] = eval.currentKey.String()
				}
			})
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not add vertex %s: %w", key, err))
				continue
			}
			if eval.currentKey != nil {
				changes.addEdge(key, *eval.currentKey)
			}
			log.Debugf("Enqueued %s", key)
			if err := eval.unevaluated.AddVertex(v); err != nil {
				errs = errors.Join(errs, err)
			}

		case err == nil:
			existing, err := eval.graph.Vertex(key)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not get existing vertex %s: %w", key, err))
				continue
			}
			if v != existing {
				existing.UpdateFrom(v)
			}

		case err != nil:
			errs = errors.Join(errs, fmt.Errorf("could not get existing vertex %s: %w", key, err))
		}
	}
	if errs != nil {
		return errs
	}

	log = eval.Log().With("op", "deps")
	for source, targets := range changes.edges {
		if len(targets) > 0 {
			log.Debug(source)
		}
		for target := range targets {
			errs = errors.Join(errs, eval.addEdge(source, target))
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func newChanges() graphChanges {
	return graphChanges{
		nodes: make(map[Key]Vertex),
		edges: make(map[Key]set.Set[Key]),
	}
}

// addNode is a convenient lower-level add for [graphChanges.nodes]
func (changes graphChanges) addNode(v Vertex) {
	changes.nodes[v.Key()] = v
}

// addEdge is a convenient lower-level add for [graphChanges.edges]
func (changes graphChanges) addEdge(source, target Key) {
	out, ok := changes.edges[source]
	if !ok {
		out = make(set.Set[Key])
		changes.edges[source] = out
	}
	out.Add(target)
}

// addEdges is a convenient lower-level add for [graphChanges.edges] for multiple targets
func (changes graphChanges) addEdges(source Key, targets set.Set[Key]) {
	out, ok := changes.edges[source]
	if !ok {
		out = make(set.Set[Key])
		changes.edges[source] = out
	}
	out.AddFrom(targets)
}

func (changes graphChanges) AddVertexAndDeps(eval *Evaluator, v Vertex) error {
	changes.nodes[v.Key()] = v
	deps, err := v.Dependencies(eval)
	if err != nil {
		return err
	}
	changes.Merge(deps)
	return nil
}

func (changes graphChanges) Merge(other graphChanges) {
	for k, v := range other.nodes {
		changes.nodes[k] = v
	}
	for k, v := range other.edges {
		changes.addEdges(k, v)
	}
}

func (eval *Evaluator) UpdateId(oldId, newId construct.ResourceId) error {
	if oldId == newId {
		return nil
	}
	eval.Log().Infof("Updating id %s to %s", oldId, newId)

	v, err := eval.Solution.RawView().Vertex(oldId)
	if err != nil {
		return err
	}
	v.ID = newId
	err = construct.PropagateUpdatedId(eval.Solution.RawView(), oldId)
	if err != nil {
		return err
	}

	topo, err := graph.TopologicalSort(eval.graph)
	if err != nil {
		return err
	}

	var errs error

	replaceVertex := func(oldKey Key, vertex Vertex) {
		errs = errors.Join(errs,
			graph_addons.ReplaceVertex(eval.graph, oldKey, Vertex(vertex), Vertex.Key),
		)
		if _, err := eval.unevaluated.Vertex(oldKey); err == nil {
			errs = errors.Join(errs,
				graph_addons.ReplaceVertex(eval.unevaluated, oldKey, Vertex(vertex), Vertex.Key),
			)
		} else if !errors.Is(err, graph.ErrVertexNotFound) {
			errs = errors.Join(errs, err)
		}
	}

	for _, key := range topo {
		vertex, err := eval.graph.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		switch vertex := vertex.(type) {
		case *propertyVertex:
			if key.Ref.Resource == oldId {
				vertex.Ref.Resource = newId
				replaceVertex(key, vertex)
			}

			for edge, rules := range vertex.EdgeRules {
				if edge.Source == oldId || edge.Target == oldId {
					delete(vertex.EdgeRules, edge)
					vertex.EdgeRules[UpdateEdgeId(edge, oldId, newId)] = rules
				}
			}

		case *edgeVertex:
			if key.Edge.Source == oldId || key.Edge.Target == oldId {
				vertex.Edge = UpdateEdgeId(vertex.Edge, oldId, newId)
				replaceVertex(key, vertex)
			}
		}
	}
	if errs != nil {
		return errs
	}

	for i, keys := range eval.evaluatedOrder {
		for key := range keys {
			oldKey := key
			if key.Ref.Resource == oldId {
				key.Ref.Resource = newId
			}
			key.Edge = UpdateEdgeId(key.Edge, oldId, newId)
			if key != oldKey {
				eval.evaluatedOrder[i].Remove(oldKey)
				eval.evaluatedOrder[i].Add(key)
			}
		}
	}

	for key := range eval.errored {
		oldKey := key
		if key.Ref.Resource == oldId {
			key.Ref.Resource = newId
		}
		key.Edge = UpdateEdgeId(key.Edge, oldId, newId)
		if key != oldKey {
			eval.errored.Remove(oldKey)
			eval.errored.Add(key)
		}
	}

	return nil
}
