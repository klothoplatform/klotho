package operational_eval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type pathExpandVertex struct {
	Edge          construct.SimpleEdge
	TempGraph     construct.Graph
	Satisfication pathSatisfication
}

type pathSatisfication struct {
	// Signals if the classification is derived from the target or not
	// we need this to know how to construct the edge we are going to run expansion on if we have resource values in the classification
	asTarget       bool
	classification *string
}

type expansionInput struct {
	dep       construct.ResourceEdge
	tempGraph construct.Graph
}

func (v *pathExpandVertex) Key() Key {
	return Key{PathSatisfication: v.Satisfication, Edge: v.Edge}
}

func (v *pathExpandVertex) Evaluate(eval *Evaluator) error {
	var errs error

	expansions, err := v.getExpansionsToRun(eval)
	if err != nil {
		errs = errors.Join(errs, err)
	}

	for _, expansion := range expansions {
		errs = errors.Join(errs, v.runExpansion(eval, expansion))
	}
	return errs
}

func (v *pathExpandVertex) runExpansion(eval *Evaluator, expansion expansionInput) error {
	var errs error
	resultGraph, err := path_selection.ExpandEdge(eval.Solution, expansion.dep, expansion.tempGraph)
	if err != nil {
		return fmt.Errorf("failed to evaluate path expand vertex. could not expand edge %s -> %s: %w", v.Edge.Source, v.Edge.Target, err)
	}

	adj, err := resultGraph.AdjacencyMap()
	if err != nil {
		return err
	}
	if len(adj) > 2 {
		_, err := eval.Solution.OperationalView().Edge(v.Edge.Source, v.Edge.Target)
		if err == nil {
			if err := eval.Solution.OperationalView().RemoveEdge(v.Edge.Source, v.Edge.Target); err != nil {
				return err
			}
		} else if !errors.Is(err, graph.ErrEdgeNotFound) {
			return err
		}
	} else if len(adj) == 2 {
		return eval.Solution.OperationalView().MakeEdgesOperational([]construct.Edge{{Source: expansion.dep.Source.ID, Target: expansion.dep.Target.ID}})
	}

	// Once the path is selected & expanded, first add all the resources to the graph
	resources := []*construct.Resource{}
	for pathId := range adj {
		if pathId.Provider == path_selection.SERVICE_API_PROVIDER {
			continue
		}
		res, err := eval.Solution.OperationalView().Vertex(pathId)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			res = construct.CreateResource(pathId)
			// add the resource to the raw view because we want to wait until after the edges are added to make it operational
			errs = errors.Join(errs, eval.Solution.RawView().AddVertex(res))

		case err != nil:
			errs = errors.Join(errs, err)
		}
		resources = append(resources, res)
	}
	if errs != nil {
		return errs
	}

	// After all the resources, then add all the dependencies
	pstr := []string{}
	edges := []construct.Edge{}
	for _, edgeMap := range adj {
		for _, edge := range edgeMap {

			if edge.Source.Provider == path_selection.SERVICE_API_PROVIDER || edge.Target.Provider == path_selection.SERVICE_API_PROVIDER {
				continue
			}
			pstr = append(pstr, fmt.Sprintf("%s -> %s", edge.Source, edge.Target))
			errs = errors.Join(errs, eval.Solution.RawView().AddEdge(edge.Source, edge.Target))
			edges = append(edges, edge)
		}
	}

	if errs != nil {
		return errs
	}
	if v.Satisfication.classification != nil {
		zap.S().Infof("Satisfied %s for %s -> %s through %s", *v.Satisfication.classification, v.Edge.Source,
			v.Edge.Target, strings.Join(pstr, " -> "))
	} else {
		zap.S().Infof("Satisfied %s -> %s through %s", v.Edge.Source, v.Edge.Target, strings.Join(pstr, ", "))
	}

	if err := eval.AddResources(resources...); err != nil {
		return err
	}

	return eval.AddEdges(edges...)
}

func (v *pathExpandVertex) getExpansionsToRun(eval *Evaluator) ([]expansionInput, error) {
	var errs error
	sourceRes, err := eval.Solution.RawView().Vertex(v.Edge.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate path expand vertex. could not find source resource %s: %w", v.Edge.Source, err)
	}
	targetRes, err := eval.Solution.RawView().Vertex(v.Edge.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate path expand vertex. could not find target resource %s: %w", v.Edge.Target, err)
	}
	edge := construct.ResourceEdge{Source: sourceRes, Target: targetRes}
	expansions := []expansionInput{}
	if v.Satisfication.classification != nil && strings.Contains(*v.Satisfication.classification, "#") {
		parts := strings.Split(*v.Satisfication.classification, "#")

		resources := []construct.ResourceId{v.Edge.Source}
		if v.Satisfication.asTarget {
			resources = []construct.ResourceId{v.Edge.Target}
		}

		for i, part := range parts {
			fieldValueResources := []construct.ResourceId{}
			if i == 0 {
				continue
			}
			for _, resId := range resources {
				r, err := eval.Solution.RawView().Vertex(resId)
				if r == nil || err != nil {
					errs = errors.Join(errs, fmt.Errorf("failed to evaluate path expand vertex. could not find resource %s: %w", resId, err))
					continue
				}
				val, err := r.GetProperty(part)
				if err != nil || val == nil {
					errs = errors.Join(errs, fmt.Errorf("failed to evaluate path expand vertex. could not find property %s on resource %s: %w", part, resId, err))
				}
				if id, ok := val.(construct.ResourceId); ok {
					fieldValueResources = append(fieldValueResources, id)
				} else if ids, ok := val.([]interface{}); ok {
					for _, idVal := range ids {
						if id, ok := idVal.(construct.ResourceId); ok {
							fieldValueResources = append(fieldValueResources, id)
						} else {
							errs = errors.Join(errs, fmt.Errorf("failed to evaluate path expand vertex. property %s on resource %s is not a resource id", part, resId))
						}
					}
				} else {
					errs = errors.Join(errs, fmt.Errorf("failed to evaluate path expand vertex. property %s on resource %s is not a resource id", part, resId))
				}
			}
			resources = fieldValueResources
		}
		for _, resId := range resources {
			res, err := eval.Solution.RawView().Vertex(resId)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to evaluate path expand vertex. could not find resource %s: %w", resId, err))
				continue
			}
			simple := construct.SimpleEdge{Source: res.ID, Target: v.Edge.Target}
			e := construct.ResourceEdge{Source: res, Target: targetRes}
			if v.Satisfication.asTarget {
				simple = construct.SimpleEdge{Source: v.Edge.Source, Target: res.ID}
				e = construct.ResourceEdge{Source: sourceRes, Target: res}
			}
			tempGraph, err := path_selection.BuildPathSelectionGraph(simple, eval.Solution.KnowledgeBase(), &parts[0])
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			expansions = append(expansions, expansionInput{dep: e, tempGraph: tempGraph})
		}
	} else {
		expansions = append(expansions, expansionInput{dep: edge, tempGraph: v.TempGraph})
	}
	return expansions, errs
}

func (v *pathExpandVertex) UpdateFrom(other Vertex) {
	otherVertex := other.(*pathExpandVertex)
	v.TempGraph = otherVertex.TempGraph
}

func (v *pathExpandVertex) Dependencies(
	ctx solution_context.SolutionContext,
) (set.Set[construct.PropertyRef], graphStates, error) {
	deps := make(set.Set[construct.PropertyRef])
	cfgCtx := solution_context.DynamicCtx(ctx)

	addDepsFromProps := func(
		res construct.ResourceId,
		dependencies []construct.ResourceId,
		skipCheckRes construct.ResourceId,
	) error {
		tmpl, err := cfgCtx.KB().GetResourceTemplate(res)
		if err != nil {
			return err
		}
		var errs error
		for k, prop := range tmpl.Properties {
			pt, err := prop.PropertyType()
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			if prop.OperationalRule != nil {
				opCtx := operational_rule.OperationalRuleContext{
					Property: prop,
					Solution: ctx,
					Data:     knowledgebase.DynamicValueData{Resource: res},
				}
				canRun, err := opCtx.EvaluateIfCondition(*prop.OperationalRule)
				if err != nil || !canRun {
					continue
				}
			}

			resType, ok := pt.(*knowledgebase.ResourcePropertyType)
			if !ok {
				listType, ok := pt.(*knowledgebase.ListPropertyType)
				if !ok || listType.Value == "" {
					continue
				}
				pType := knowledgebase.Property{Type: listType.Value}
				pt, err = pType.PropertyType()
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
				resType, ok = pt.(*knowledgebase.ResourcePropertyType)
				if !ok {
					continue
				}
			}
			for _, dep := range dependencies {
				if resType.Value.Matches(dep) && !skipCheckRes.Matches(dep) {
					deps.Add(construct.PropertyRef{Resource: res, Property: k})
					break
				}
			}
		}
		return errs
	}

	var errs error
	if v.Satisfication.classification != nil {
		parts := strings.Split(*v.Satisfication.classification, "#")
		if len(parts) >= 2 {
			currResources := []construct.ResourceId{v.Edge.Source}
			if v.Satisfication.asTarget {
				currResources = []construct.ResourceId{v.Edge.Target}
			}
			for i, part := range parts {
				if i == 0 || part == "" {
					continue
				}
				var nextResources []construct.ResourceId
				for _, currResource := range currResources {
					val, err := cfgCtx.FieldValue(part, currResource)
					if err != nil {
						deps.Add(construct.PropertyRef{Resource: currResource, Property: part})
						continue
					}
					if id, ok := val.(construct.ResourceId); ok {
						nextResources = append(nextResources, id)
					} else if ids, ok := val.([]construct.ResourceId); ok {
						nextResources = append(nextResources, ids...)
					}
				}
				currResources = nextResources
			}
		}
	}

	// if we have a temp graph we can analyze the paths in it for possible dependencies on property vertices
	// if we dont, we should return what we currently have
	if v.TempGraph == nil {
		return deps, nil, nil
	}

	srcDeps, err := construct.AllDownstreamDependencies(v.TempGraph, v.Edge.Source)
	if err != nil {
		return deps, nil, err
	}
	errs = errors.Join(errs, addDepsFromProps(v.Edge.Source, srcDeps, v.Edge.Target))

	targetDeps, err := construct.AllUpstreamDependencies(v.TempGraph, v.Edge.Target)
	if err != nil {
		return deps, nil, err
	}
	errs = errors.Join(errs, addDepsFromProps(v.Edge.Target, targetDeps, v.Edge.Source))
	return deps, nil, errs
}

func (eval *Evaluator) AddPath(source, target construct.ResourceId) (err error) {

	generateAndAddVertex := func(
		vs verticesAndDeps,
		edge construct.SimpleEdge,
		kb knowledgebase.TemplateKB,
		satisfication pathSatisfication) error {
		var tempGraph construct.Graph
		if satisfication.classification == nil || !strings.Contains(*satisfication.classification, "#") {
			tempGraph, err = path_selection.BuildPathSelectionGraph(edge, kb, satisfication.classification)
			if err != nil {
				return err
			}
		}
		vertex := pathExpandVertex{Edge: edge, Satisfication: satisfication, TempGraph: tempGraph}
		return vs.AddDependencies(eval.Solution, &vertex)
	}

	kb := eval.Solution.KnowledgeBase()
	srcTempalte, err := kb.GetResourceTemplate(source)
	if err != nil {
		return err
	}
	targetTemplate, err := kb.GetResourceTemplate(target)
	if err != nil {
		return err
	}
	edge := construct.SimpleEdge{Source: source, Target: target}
	pathSatisfications := []pathSatisfication{}
	for _, src := range srcTempalte.PathSatisfaction.AsSource {
		srcString := src
		pathSatisfications = append(pathSatisfications, pathSatisfication{asTarget: false, classification: &srcString})
	}
	for _, trgt := range targetTemplate.PathSatisfaction.AsTarget {
		trgtString := trgt
		pathSatisfications = append(pathSatisfications, pathSatisfication{asTarget: true, classification: &trgtString})
	}

	vs := make(verticesAndDeps)
	var errs error
	for _, satisfication := range pathSatisfications {
		errs = errors.Join(errs, generateAndAddVertex(vs, edge, kb, satisfication))
	}
	if len(pathSatisfications) == 0 {
		errs = errors.Join(errs, generateAndAddVertex(vs, edge, kb, pathSatisfication{}))
	}
	return errors.Join(errs, eval.enqueue(vs))
}
