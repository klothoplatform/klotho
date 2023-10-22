package property_eval

import (
	"errors"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type (
	PropertyEval struct {
		Solution solution_context.SolutionContext

		// graph holds all of the property dependencies regardless of whether they've been evaluated or not
		graph Graph

		unevaluated Graph

		evaluatedOrder [][]EvaluationKey
	}

	EvaluationKey struct {
		Ref        construct.PropertyRef
		Edge       construct.SimpleEdge
		GraphState string
	}

	EvaluationVertex interface {
		Key() EvaluationKey
		Evaluate(eval *PropertyEval) error
		UpdateFrom(other EvaluationVertex)
		Dependencies(cfgCtx knowledgebase.DynamicValueContext) (set.Set[construct.PropertyRef], graphStates, error)
	}

	verticesAndDeps map[EvaluationVertex]set.Set[EvaluationKey]

	graphStates map[string]func(construct.Graph) (bool, error)
)

func NewPropertyEval(ctx solution_context.SolutionContext) *PropertyEval {
	return &PropertyEval{
		Solution:    ctx,
		graph:       newGraph(),
		unevaluated: newGraph(),
	}
}

func (key EvaluationKey) String() string {
	if !key.Ref.Resource.IsZero() {
		return key.Ref.String()
	}
	if key.GraphState != "" {
		return key.GraphState
	}
	return key.Edge.String()
}

func (eval *PropertyEval) enqueue(vs verticesAndDeps) error {
	var errs error
	for v, deps := range vs {
		key := v.Key()
		_, err := eval.graph.Vertex(key)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			err := eval.graph.AddVertex(v)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			zap.S().With("op", "enqueue").Debugf("Enqueued %s (%d deps)", v.Key(), len(deps))
			if err := eval.unevaluated.AddVertex(v); err != nil {
				errs = errors.Join(errs, err)
			}

		case err == nil:
			existing, err := eval.graph.Vertex(v.Key())
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			if v != existing {
				zap.S().With("op", "enqueue").Debugf("Updating %s (%d deps)", v.Key(), len(deps))
				existing.UpdateFrom(v)
			}

		case err != nil:
			errs = errors.Join(errs, err)
		}
	}
	if errs != nil {
		return errs
	}
	for source, targets := range vs {
		zap.S().With("op", "deps").Debug(source.Key())
		for target := range targets {
			err := eval.graph.AddEdge(source.Key(), target)
			if err != nil {
				// NOTE(gg): If this fails with target not in graph, then we might need to add the target in with a
				// new vertex type of "overwrite me later".
				errs = errors.Join(errs, err)
			}
			_, err = eval.unevaluated.Vertex(target)
			switch {
			case errors.Is(err, graph.ErrVertexNotFound):
				// the 'graph.AddEdge' succeeded, thus the target exists in the total graph
				// which means that the target vertex is done, so don't add the edge
				zap.S().With("op", "deps").Debugf("  -> %s (done)", target)

			case err != nil:
				errs = errors.Join(errs, err)

			default:
				zap.S().With("op", "deps").Debugf("  -> %s", target)
				errs = errors.Join(errs, eval.unevaluated.AddEdge(source.Key(), target))
			}
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func (vs *verticesAndDeps) AddAll(other verticesAndDeps) {
	if *vs == nil {
		*vs = make(verticesAndDeps)
	}
	for v, deps := range other {
		if (*vs)[v] == nil {
			(*vs)[v] = make(set.Set[EvaluationKey])
		}
		(*vs)[v].AddFrom(deps)
	}
}

func (vs *verticesAndDeps) AddRefs(k EvaluationVertex, deps set.Set[construct.PropertyRef]) {
	if *vs == nil {
		*vs = make(verticesAndDeps)
	}
	if (*vs)[k] == nil {
		(*vs)[k] = make(set.Set[EvaluationKey])
	}
	for dep := range deps {
		(*vs)[k].Add(EvaluationKey{Ref: dep})
	}
}

func (vs *verticesAndDeps) AddGraphStates(k EvaluationVertex, states graphStates) {
	if *vs == nil {
		*vs = make(verticesAndDeps)
	}
	for repr, test := range states {
		v := &graphStateVertex{repr: repr, Test: test}
		(*vs)[v] = make(set.Set[EvaluationKey])
		(*vs)[k].Add(v.Key())
	}
}

func (vs *verticesAndDeps) AddDependencies(cfgCtx knowledgebase.DynamicValueContext, v EvaluationVertex) error {
	deps, gs, err := v.Dependencies(cfgCtx)
	vs.AddRefs(v, deps)
	vs.AddGraphStates(v, gs)
	return err
}

func VertexLess(a, b EvaluationKey) bool {
	if a.Ref.Resource.IsZero() != b.Ref.Resource.IsZero() {
		// sort properties before edges
		return a.Ref.Resource.IsZero()
	}
	if a.Ref.Resource.IsZero() {
		// both are edges, sort by source first then by target
		if a.Edge.Source != b.Edge.Source {
			return construct.ResourceIdLess(a.Edge.Source, b.Edge.Source)
		}
		return construct.ResourceIdLess(a.Edge.Target, b.Edge.Target)
	}

	// both are properties
	if a.Ref.Resource != b.Ref.Resource {
		return construct.ResourceIdLess(a.Ref.Resource, b.Ref.Resource)
	}
	return a.Ref.Property < b.Ref.Property
}

func (eval *PropertyEval) UpdateId(oldId, newId construct.ResourceId) error {
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

	topo, err := graph.StableTopologicalSort(eval.graph, VertexLess)
	if err != nil {
		return err
	}

	var errs error

	replaceVertex := func(oldKey EvaluationKey, vertex EvaluationVertex) {
		errs = errors.Join(errs,
			graph_addons.ReplaceVertex(eval.graph, oldKey, EvaluationVertex(vertex), EvaluationVertex.Key),
		)
		if _, err := eval.unevaluated.Vertex(oldKey); err == nil {
			errs = errors.Join(errs,
				graph_addons.ReplaceVertex(eval.unevaluated, oldKey, EvaluationVertex(vertex), EvaluationVertex.Key),
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
		for j, key := range keys {
			if key.Ref.Resource == oldId {
				key.Ref.Resource = newId
			}
			key.Edge = UpdateEdgeId(key.Edge, oldId, newId)
			eval.evaluatedOrder[i][j] = key
		}
	}

	return nil
}
