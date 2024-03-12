package operational_eval

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/path_selection"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type Graph graph.Graph[Key, Vertex]

func newGraph(log *zap.Logger) Graph {
	g := graph.NewWithStore(
		Vertex.Key,
		graph_addons.NewMemoryStore[Key, Vertex](),
		graph.Directed(),
		graph.PreventCycles(),
	)
	if log != nil {
		g = graph_addons.LoggingGraph[Key, Vertex]{
			Graph: g,
			Log:   log.Sugar(),
			Hash:  Vertex.Key,
		}
	}
	return g
}

func (eval *Evaluator) AddResources(rs ...*construct.Resource) error {
	changes := newChanges()
	var errs error
	for _, res := range rs {
		tmpl, err := eval.Solution.KnowledgeBase().GetResourceTemplate(res.ID)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		rvs, err := eval.resourceVertices(res, tmpl)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not add resource eval vertices %s: %w", res.ID, err))
			continue
		}
		changes.Merge(rvs)
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(changes)
}

func (eval *Evaluator) AddEdges(es ...construct.Edge) error {
	changes := newChanges()
	var errs error
	for _, e := range es {
		tmpl := eval.Solution.KnowledgeBase().GetEdgeTemplate(e.Source, e.Target)
		var evs graphChanges
		var err error
		if tmpl == nil {
			evs, err = eval.pathVertices(e.Source, e.Target)
		} else {
			evs, err = eval.edgeVertices(e, tmpl)
		}
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not add edge eval vertex %s -> %s: %w", e.Source, e.Target, err))
			continue
		}
		changes.Merge(evs)
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(changes)
}

func (eval *Evaluator) pathVertices(source, target construct.ResourceId) (graphChanges, error) {
	changes := newChanges()

	generateAndAddVertex := func(
		edge construct.SimpleEdge,
		kb knowledgebase.TemplateKB,
		satisfication knowledgebase.EdgePathSatisfaction,
	) error {
		if satisfication.Classification == "" {
			return fmt.Errorf("edge %s has no classification to expand", edge)
		}

		buildTempGraph := true
		// We are checking to see if either of the source or target nodes will change due to property references,
		// if there are property references we want to ensure the correct dependency ordering is in place so
		// we cannot yet split the expansion vertex up or build the temp graph
		if satisfication.Source.PropertyReferenceChangesBoundary() || satisfication.Target.PropertyReferenceChangesBoundary() {
			buildTempGraph = false
		}

		var tempGraph construct.Graph
		if buildTempGraph {
			var err error
			tempGraph, err = path_selection.BuildPathSelectionGraph(edge, kb, satisfication.Classification)
			if err != nil {
				return fmt.Errorf("could not build temp graph for %s: %w", edge, err)
			}
		}
		vertex := &pathExpandVertex{SatisfactionEdge: edge, Satisfication: satisfication, TempGraph: tempGraph}
		return changes.AddVertexAndDeps(eval, vertex)
	}

	kb := eval.Solution.KnowledgeBase()

	edge := construct.SimpleEdge{Source: source, Target: target}
	pathSatisfications, err := kb.GetPathSatisfactionsFromEdge(source, target)
	if err != nil {
		return changes, fmt.Errorf("could not get path satisfications for %s: %w", edge, err)
	}

	var errs error
	for _, satisfication := range pathSatisfications {
		errs = errors.Join(errs, generateAndAddVertex(edge, kb, satisfication))
	}
	if len(pathSatisfications) == 0 {
		errs = errors.Join(errs, fmt.Errorf("could not find any path satisfications for %s", edge))
	}
	return changes, errs
}

func UpdateEdgeId(e construct.SimpleEdge, oldId, newId construct.ResourceId) construct.SimpleEdge {
	switch {
	case e.Source == oldId:
		e.Source = newId
	case e.Target == oldId:
		e.Target = newId
	}
	return e
}

func (eval *Evaluator) resourceVertices(
	res *construct.Resource,
	tmpl *knowledgebase.ResourceTemplate,
) (graphChanges, error) {
	changes := newChanges()
	var errs error
	addProp := func(prop knowledgebase.Property) error {
		vertex := &propertyVertex{
			Ref:            construct.PropertyRef{Resource: res.ID, Property: prop.Details().Path},
			Template:       prop,
			EdgeRules:      make(map[construct.SimpleEdge][]knowledgebase.OperationalRule),
			TransformRules: make(map[construct.SimpleEdge]*set.HashedSet[string, knowledgebase.OperationalRule]),
		}

		errs = errors.Join(errs, changes.AddVertexAndDeps(eval, vertex))
		return nil
	}
	errs = errors.Join(errs, tmpl.LoopProperties(res, addProp))

	for _, rule := range tmpl.AdditionalRules {
		vertex := &resourceRuleVertex{
			Resource: res.ID,
			Rule:     rule,
			hash:     rule.Hash(),
		}
		errs = errors.Join(errs, changes.AddVertexAndDeps(eval, vertex))
	}

	return changes, errs
}

func (eval *Evaluator) edgeVertices(
	edge construct.Edge,
	tmpl *knowledgebase.EdgeTemplate,
) (graphChanges, error) {
	changes := newChanges()

	opVertex := edgeVertexWithRules(
		construct.SimpleEdge{Source: edge.Source, Target: edge.Target},
		tmpl.OperationalRules,
	)

	return changes, changes.AddVertexAndDeps(eval, opVertex)
}

func (eval *Evaluator) removeKey(k Key) error {
	err := graph_addons.RemoveVertexAndEdges(eval.unevaluated, k)
	if err == nil || errors.Is(err, graph.ErrVertexNotFound) {
		return graph_addons.RemoveVertexAndEdges(eval.graph, k)
	}
	return err
}

func (eval *Evaluator) RemoveEdge(source, target construct.ResourceId) error {
	g := eval.graph
	edge := construct.SimpleEdge{Source: source, Target: target}

	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}

	var errs error
	checkStates := make(set.Set[Key])
	for key := range pred {
		v, err := g.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not get vertex for %s: %w", key, err))
			continue
		}

		switch v := v.(type) {
		case *propertyVertex:
			for vEdge := range v.EdgeRules {
				if vEdge == edge {
					delete(v.EdgeRules, edge)
				}
			}

		case *edgeVertex:
			if v.Edge == edge {
				errs = errors.Join(errs, eval.removeKey(v.Key()))
			}

		case *graphStateVertex:
			checkStates.Add(v.Key())
		}
	}
	if errs != nil {
		return fmt.Errorf("could not remove edge %s: %w", edge, err)
	}

	// recompute the predecessors, since we may have removed some edges
	pred, err = g.PredecessorMap()
	if err != nil {
		return err
	}

	// Clean up any graph state keys that are no longer referenced. They don't do any harm except the performance
	// impact of recomputing the dependencies.
	for v := range checkStates {
		if len(pred[v]) == 0 {
			errs = errors.Join(errs, eval.removeKey(v))
		}
	}
	if errs != nil {
		return fmt.Errorf("could not clean up graph state keys when removing %s: %w", edge, errs)
	}
	return nil
}

// RemoveResource removes all edges from the resource. any property references (as [ResourceId] or [PropertyRef])
// to the resource, and finally the resource itself.
func (eval *Evaluator) RemoveResource(id construct.ResourceId) error {
	g := eval.graph

	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}

	var errs error
	checkStates := make(set.Set[Key])
	for key := range pred {
		v, err := g.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not get vertex for %s: %w", key, err))
			continue
		}

		switch v := v.(type) {
		case *propertyVertex:
			if v.Ref.Resource == id {
				errs = errors.Join(errs, eval.removeKey(v.Key()))
				continue
			}
			for edge := range v.EdgeRules {
				if edge.Source == id || edge.Target == id {
					delete(v.EdgeRules, edge)
				}
			}

		case *edgeVertex:
			if v.Edge.Source == id || v.Edge.Target == id {
				errs = errors.Join(errs, eval.removeKey(v.Key()))
			}

		case *graphStateVertex:
			checkStates.Add(v.Key())
		}
	}
	if errs != nil {
		return fmt.Errorf("could not remove resource %s: %w", id, errs)
	}

	// recompute the predecessors, since we may have removed some edges
	pred, err = g.PredecessorMap()
	if err != nil {
		return err
	}

	// Clean up any graph state keys that are no longer referenced. They don't do any harm except the performance
	// impact of recomputing the dependencies.
	for v := range checkStates {
		if len(pred[v]) == 0 {
			errs = errors.Join(errs, eval.removeKey(v))
		}
	}
	if errs != nil {
		return fmt.Errorf("could not clean up graph state keys when removing %s: %w", id, errs)
	}
	return nil
}
