package property_eval

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
)

type Graph graph.Graph[EvaluationKey, EvaluationVertex]

func newGraph() Graph {
	return graph.NewWithStore(
		EvaluationVertex.Key,
		graph_addons.NewMemoryStore[EvaluationKey, EvaluationVertex](),
		graph.Directed(),
		graph.PreventCycles(),
	)
}

func (eval *PropertyEval) AddResources(rs ...*construct.Resource) error {
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

func (eval *PropertyEval) AddEdges(es ...construct.Edge) error {
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

func (eval *PropertyEval) resourceVertices(
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

func (eval *PropertyEval) edgeVertices(
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
				existing, err := eval.graph.Vertex(EvaluationKey{Ref: ref})
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

func (eval *PropertyEval) RemoveEdge(source, target construct.ResourceId) error {
	g := eval.graph
	edge := construct.SimpleEdge{Source: source, Target: target}

	removeKey := func(k EvaluationKey) error {
		err := graph_addons.RemoveVertexAndEdges(g, k)
		unevalErr := graph_addons.RemoveVertexAndEdges(g, k)
		if errors.Is(unevalErr, graph.ErrVertexNotFound) {
			// if the key is already evaluated, it won't be in the unevaluated graph, thus would
			// return an [ErrVertexNotFound]. Ignore those
			unevalErr = nil
		}
		return errors.Join(err, unevalErr)
	}

	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}

	var errs error
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
				errs = errors.Join(errs, removeKey(v.Key()))
			}

		case *graphStateVertex:
			if len(pred[v.Key()]) == 0 {
				errs = errors.Join(errs, removeKey(v.Key()))
			}
		}
	}
	if errs != nil {
		return fmt.Errorf("could not remove edge %s -> %s: %w", source, target, err)
	}
	return nil
}

// RemoveResource removes all edges from the resource. any property references (as [ResourceId] or [PropertyRef])
// to the resource, and finally the resource itself.
func (eval *PropertyEval) RemoveResource(id construct.ResourceId) error {
	g := eval.graph

	pred, err := g.PredecessorMap()
	if err != nil {
		return err
	}

	removeKey := func(k EvaluationKey) error {
		err := graph_addons.RemoveVertexAndEdges(g, k)
		unevalErr := graph_addons.RemoveVertexAndEdges(g, k)
		if errors.Is(unevalErr, graph.ErrVertexNotFound) {
			// if the key is already evaluated, it won't be in the unevaluated graph, thus would
			// return an [ErrVertexNotFound]. Ignore those
			unevalErr = nil
		}
		return errors.Join(err, unevalErr)
	}

	var errs error
	checkStates := make(set.Set[EvaluationKey])
	for key := range pred {
		v, err := g.Vertex(key)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not get vertex for %s: %w", key, err))
			continue
		}

		switch v := v.(type) {
		case *propertyVertex:
			if v.Ref.Resource == id {
				errs = errors.Join(errs, removeKey(v.Key()))
				continue
			}
			for edge := range v.EdgeRules {
				if edge.Source == id || edge.Target == id {
					delete(v.EdgeRules, edge)
				}
			}

		case *edgeVertex:
			if v.Edge.Source != id && v.Edge.Target != id {
				continue
			}
			errs = errors.Join(errs, removeKey(v.Key()))

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

	for v := range checkStates {
		if len(pred[v]) == 0 {
			errs = errors.Join(errs, removeKey(v))
		}
	}
	if errs != nil {
		return fmt.Errorf("could not clean up graph state keys when removing %s: %w", id, errs)
	}
	return nil
}
