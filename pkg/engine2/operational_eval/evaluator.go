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
	}

	Key struct {
		Ref               construct.PropertyRef
		Edge              construct.SimpleEdge
		GraphState        graphStateRepr
		PathSatisfication *knowledgebase.EdgePathSatisfaction
	}

	Vertex interface {
		Key() Key
		Evaluate(eval *Evaluator) error
		UpdateFrom(other Vertex)
		Dependencies(ctx solution_context.SolutionContext) (vertexDependencies, error)
	}

	vertexDependencies struct {
		Outgoing    set.Set[Key]
		Incoming    set.Set[Key]
		GraphStates []*graphStateVertex
	}

	graphChanges struct {
		nodes set.Set[Vertex]
		// edges is map[source]targets
		edges map[Key]set.Set[Key]
	}
)

func NewEvaluator(ctx solution_context.SolutionContext) *Evaluator {
	return &Evaluator{
		Solution:    ctx,
		graph:       newGraph(),
		unevaluated: newGraph(),
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
			key.PathSatisfication.Classification,
		}
		if key.PathSatisfication.AsTarget {
			args = append(args, "target")
		}
		return fmt.Sprintf("Expand(%s)", strings.Join(args, ", "))
	}
	if key.Edge != (construct.SimpleEdge{}) {
		return key.Edge.String()
	}
	return "<empty>"
}

func (eval *Evaluator) enqueue(changes graphChanges) error {
	var errs error
	for v := range changes.nodes {
		key := v.Key()
		_, err := eval.graph.Vertex(key)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			err := eval.graph.AddVertex(v)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not add vertex %s: %w", key, err))
				continue
			}
			zap.S().With("op", "enqueue").Debugf("Enqueued %s", key)
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
				zap.S().With("op", "enqueue").Debugf("Updating %s", key)
				existing.UpdateFrom(v)
			}

		case err != nil:
			errs = errors.Join(errs, fmt.Errorf("could not get existing vertex %s: %w", key, err))
		}
	}
	if errs != nil {
		return errs
	}
	for source, targets := range changes.edges {
		zap.S().With("op", "deps").Debug(source)
		for target := range targets {
			err := eval.graph.AddEdge(source, target)
			if err != nil {
				// NOTE(gg): If this fails with target not in graph, then we might need to add the target in with a
				// new vertex type of "overwrite me later". It would be an odd situation though, which is why it is
				// an error for now.
				errs = errors.Join(errs, fmt.Errorf("could not add edge %s -> %s: %w", source, target, err))
				continue
			}
			_, err = eval.unevaluated.Vertex(target)
			switch {
			case errors.Is(err, graph.ErrVertexNotFound):
				// the 'graph.AddEdge' succeeded, thus the target exists in the total graph
				// which means that the target vertex is done, so ignore adding the edge to the unevaluated graph
				zap.S().With("op", "deps").Debugf("  -> %s (done)", target)

			case err != nil:
				errs = errors.Join(errs, fmt.Errorf("could not get unevaluated vertex %s: %w", target, err))

			default:
				zap.S().With("op", "deps").Debugf("  -> %s", target)
				err := eval.unevaluated.AddEdge(source, target)
				if err != nil {
					errs = errors.Join(errs, fmt.Errorf("could not add unevaluated edge %s -> %s: %w", source, target, err))
				}
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func newDeps() vertexDependencies {
	return vertexDependencies{
		Outgoing: make(set.Set[Key]),
		Incoming: make(set.Set[Key]),
	}
}

func newChanges() graphChanges {
	return graphChanges{
		nodes: make(set.Set[Vertex]),
		edges: make(map[Key]set.Set[Key]),
	}
}

func (changes graphChanges) AddVertex(sol solution_context.SolutionContext, v Vertex) error {
	changes.nodes.Add(v)
	deps, err := v.Dependencies(sol)
	if err != nil {
		return err
	}
	vKey := v.Key()
	out, ok := changes.edges[vKey]
	if !ok {
		out = make(set.Set[Key])
		changes.edges[vKey] = out
	}
	out.AddFrom(deps.Outgoing)

	for in := range deps.Incoming {
		incoming, ok := changes.edges[in]
		if !ok {
			incoming = make(set.Set[Key])
			changes.edges[in] = incoming
		}
		incoming.Add(vKey)
	}

	for _, state := range deps.GraphStates {
		changes.nodes.Add(state)
	}

	return nil
}

func (changes graphChanges) Merge(other graphChanges) {
	changes.nodes.AddFrom(other.nodes)
	for k, v := range other.edges {
		out, ok := changes.edges[k]
		if !ok {
			out = make(set.Set[Key])
			changes.edges[k] = out
		}
		out.AddFrom(v)
	}
}

func (eval *Evaluator) UpdateId(oldId, newId construct.ResourceId) error {
	if oldId == newId {
		return nil
	}
	zap.S().Infof("Updating id %s to %s", oldId, newId)

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

	return nil
}
