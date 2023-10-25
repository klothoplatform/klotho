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

type Graph graph.Graph[Key, Vertex]

func newGraph() Graph {
	g := graph.NewWithStore(
		Vertex.Key,
		graph_addons.NewMemoryStore[Key, Vertex](),
		graph.Directed(),
		graph.PreventCycles(),
	)
	g = graph_addons.LoggingGraph[Key, Vertex]{
		Graph: g,
		Log:   zap.L().With(zap.String("graph", "evaluation")).Sugar(),
		Hash:  Vertex.Key,
	}
	return g
}

func (eval *Evaluator) AddResources(rs ...*construct.Resource) error {
	vs := make(verticesAndDeps)
	var errs error
	for _, res := range rs {
		tmpl, err := eval.Solution.KnowledgeBase().GetResourceTemplate(res.ID)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		rvs, err := eval.resourceVertices(res, tmpl)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		vs.AddAll(rvs)
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(vs)
}

func (eval *Evaluator) AddEdges(es ...construct.Edge) error {
	vs := make(verticesAndDeps)
	var errs error
	for _, e := range es {
		tmpl := eval.Solution.KnowledgeBase().GetEdgeTemplate(e.Source, e.Target)
		if tmpl == nil {
			continue
		}
		evs, err := eval.edgeVertices(e, tmpl)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		vs.AddAll(evs)
	}
	if errs != nil {
		return errs
	}
	return eval.enqueue(vs)
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
) (verticesAndDeps, error) {
	vs := make(verticesAndDeps)
	var errs error

	cfgCtx := solution_context.DynamicCtx(eval.Solution)

	queue := []knowledgebase.Properties{tmpl.Properties}
	var props knowledgebase.Properties
	for len(queue) > 0 {
		props, queue = queue[0], queue[1:]
		for _, prop := range props {
			vertex := &propertyVertex{
				Ref:       construct.PropertyRef{Resource: res.ID, Property: prop.Path},
				Template:  prop,
				EdgeRules: make(map[construct.SimpleEdge][]knowledgebase.OperationalRule),
			}

			err := vs.AddDependencies(cfgCtx, vertex)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}

			if strings.HasPrefix(prop.Type, "list") || strings.HasPrefix(prop.Type, "set") {
				// Because lists/sets will start as empty, do not recurse into their sub-properties
				continue
			}
			if prop.Properties != nil {
				queue = append(queue, prop.Properties)
			}
		}
	}
	return vs, errs
}

func (eval *Evaluator) edgeVertices(
	edge construct.Edge,
	tmpl *knowledgebase.EdgeTemplate,
) (verticesAndDeps, error) {
	vs := make(verticesAndDeps)
	var errs error

	cfgCtx := solution_context.DynamicCtx(eval.Solution)
	data := knowledgebase.DynamicValueData{Edge: &edge}
	resEdge := construct.SimpleEdge{Source: edge.Source, Target: edge.Target}

	opVertex := &edgeVertex{Edge: resEdge}

	vertices := make(map[construct.PropertyRef]*propertyVertex)
	for i, rule := range tmpl.OperationalRules {
		if len(rule.Steps) > 0 {
			opVertex.Rules = append(opVertex.Rules, knowledgebase.OperationalRule{
				// Split out only the steps, the configuration goes to the property it references
				If: rule.If, Steps: rule.Steps,
			})
		}

		for j, config := range rule.ConfigurationRules {
			var ref construct.PropertyRef
			err := cfgCtx.ExecuteDecode(config.Resource, data, &ref.Resource)
			if err != nil {
				errs = errors.Join(errs,
					fmt.Errorf("could not execute resource template for edge rule [%d][%d]: %w", i, j, err),
				)
			}
			err = cfgCtx.ExecuteDecode(config.Config.Field, data, &ref.Property)
			if err != nil {
				errs = errors.Join(errs,
					fmt.Errorf("could not execute config.field template for edge rule [%d][%d]: %w", i, j, err),
				)
			}
			vertex, ok := vertices[ref]
			if !ok {
				existing, err := eval.graph.Vertex(Key{Ref: ref})
				switch {
				case errors.Is(err, graph.ErrVertexNotFound):
					vertex = &propertyVertex{Ref: ref, EdgeRules: make(map[construct.SimpleEdge][]knowledgebase.OperationalRule)}
				case err != nil:
					errs = errors.Join(errs, fmt.Errorf("could not attempt to get existing vertex for %s: %w", ref, err))
					continue
				default:
					if v, ok := existing.(*propertyVertex); ok {
						vertex = v
					} else {
						errs = errors.Join(errs, fmt.Errorf("existing vertex for %s is not a property vertex", ref))
						continue
					}
				}
				vertices[ref] = vertex
			}

			vertex.EdgeRules[resEdge] = append(vertex.EdgeRules[resEdge], knowledgebase.OperationalRule{
				If:                 rule.If,
				ConfigurationRules: []knowledgebase.ConfigurationRule{config},
			})
		}
	}
	if errs != nil {
		return nil, errs
	}

	if len(opVertex.Rules) > 0 {
		errs = errors.Join(errs, vs.AddDependencies(cfgCtx, opVertex))
	}

	// do this in a second pass so that edges config that reference the same property (rare, but possible)
	// will get batched before calling [depsForProp].
	for _, vertex := range vertices {
		errs = errors.Join(errs, vs.AddDependencies(cfgCtx, vertex))
	}

	return vs, errs
}

func (eval *Evaluator) removeKey(k Key) error {
	err := graph_addons.RemoveVertexAndEdges(eval.unevaluated, k)
	if err == nil {
		return graph_addons.RemoveVertexAndEdges(eval.graph, k)
	} else if errors.Is(err, graph.ErrVertexNotFound) {
		// the key was already evaluated, leave it in the graph for debugging purposes
		return nil
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
