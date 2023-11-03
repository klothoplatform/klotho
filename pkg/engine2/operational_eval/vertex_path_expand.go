package operational_eval

import (
	"errors"
	"fmt"
	"reflect"
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
	Satisfication knowledgebase.EdgePathSatisfaction
}

type ExpansionInput struct {
	Dep            construct.ResourceEdge
	Classification string
	TempGraph      construct.Graph
}

func (v *pathExpandVertex) Key() Key {
	return Key{PathSatisfication: &v.Satisfication, Edge: v.Edge}
}

func (v *pathExpandVertex) Evaluate(eval *Evaluator) error {
	var errs error

	expansions, err := v.getExpansionsToRun(eval)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("could not get expansions to run: %w", err))
	}

	for _, expansion := range expansions {
		errs = errors.Join(errs, v.runExpansion(eval, expansion))
	}
	return errs
}

func expansionResultString(result construct.Graph, dep construct.ResourceEdge) (string, error) {
	sb := new(strings.Builder)
	handled := make(set.Set[construct.SimpleEdge])

	path, err := graph.ShortestPath(result, dep.Source.ID, dep.Target.ID)
	if err != nil {
		return "", fmt.Errorf("expansion result does not contain path from %s to %s: %w", dep.Source, dep.Target, err)
	}
	for i, res := range path {
		if i == 0 {
			sb.WriteString(res.String())
			continue
		}
		fmt.Fprintf(sb, " -> %s", res)
		handled.Add(construct.SimpleEdge{Source: path[i-1], Target: res})
	}

	edges, err := result.Edges()
	if err != nil {
		return sb.String(), err
	}

	for _, e := range edges {
		se := construct.SimpleEdge{Source: e.Source, Target: e.Target}
		if handled.Contains(se) {
			continue
		}
		fmt.Fprintf(sb, ", %s", se.String())
	}

	return sb.String(), nil
}

func (v *pathExpandVertex) runExpansion(eval *Evaluator, expansion ExpansionInput) error {
	var errs error
	resultGraph, err := path_selection.ExpandEdge(eval.Solution, expansion.Dep, expansion.TempGraph)
	if err != nil {
		return fmt.Errorf("failed to evaluate path expand vertex. could not expand edge %s: %w", v.Edge, err)
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
		err = eval.Solution.RawView().AddEdge(expansion.Dep.Source.ID, expansion.Dep.Target.ID)
		if err != nil {
			return err
		}
		return eval.Solution.OperationalView().MakeEdgesOperational([]construct.Edge{
			{Source: expansion.Dep.Source.ID, Target: expansion.Dep.Target.ID},
		})
	}

	// Once the path is selected & expanded, first add all the resources to the graph
	resources := []*construct.Resource{}
	for pathId := range adj {
		res, err := eval.Solution.OperationalView().Vertex(pathId)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			res = construct.CreateResource(pathId)
			// add the resource to the raw view because we want to wait until after the edges are added to make it operational
			errs = errors.Join(errs, eval.Solution.OperationalView().AddVertex(res))

		case err != nil:
			errs = errors.Join(errs, err)
		}
		resources = append(resources, res)
	}
	if errs != nil {
		return errs
	}

	resultStr, err := expansionResultString(resultGraph, expansion.Dep)
	if err != nil {
		return err
	}

	// After all the resources, then add all the dependencies
	edges := []construct.Edge{}
	for _, edgeMap := range adj {
		for _, edge := range edgeMap {
			errs = errors.Join(errs, eval.Solution.RawView().AddEdge(edge.Source, edge.Target))
			edges = append(edges, edge)
		}
	}

	if errs != nil {
		return errs
	}
	if v.Satisfication.Classification != "" {
		zap.S().Infof("Satisfied %s for %s through %s", v.Satisfication.Classification, v.Edge, resultStr)
	} else {
		zap.S().Infof("Satisfied %s -> %s through %s", v.Edge.Source, v.Edge.Target, resultStr)
	}

	if err := eval.AddResources(resources...); err != nil {
		return err
	}

	return eval.AddEdges(edges...)
}

func (v *pathExpandVertex) getExpansionsToRun(eval *Evaluator) ([]ExpansionInput, error) {
	var result []ExpansionInput
	var errs error
	sourceRes, err := eval.Solution.RawView().Vertex(v.Edge.Source)
	if err != nil {
		return nil, fmt.Errorf("could not find source resource %s: %w", v.Edge.Source, err)
	}
	targetRes, err := eval.Solution.RawView().Vertex(v.Edge.Target)
	if err != nil {
		return nil, fmt.Errorf("could not find target resource %s: %w", v.Edge.Target, err)
	}
	edge := construct.ResourceEdge{Source: sourceRes, Target: targetRes}
	expansions, err := DeterminePathSatisfactionInputs(eval.Solution, v.Satisfication, edge)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	for _, expansion := range expansions {
		if expansion.Dep.Source != edge.Source || expansion.Dep.Target != edge.Target {
			simple := construct.SimpleEdge{Source: expansion.Dep.Source.ID, Target: expansion.Dep.Target.ID}
			tempGraph, err := path_selection.BuildPathSelectionGraph(simple, eval.Solution.KnowledgeBase(), expansion.Classification)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("error getting expansions to run. could not build path selection graph: %w", err))
				continue
			}
			temp, err := tempGraph.Clone()
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("error getting expansions to run. could not clone path selection graph: %w", err))
				continue
			}
			result = append(result, ExpansionInput{Dep: expansion.Dep, Classification: expansion.Classification, TempGraph: temp})
		} else {
			result = append(result, ExpansionInput{Dep: expansion.Dep, Classification: expansion.Classification, TempGraph: v.TempGraph})
		}
	}
	return result, errs
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
			if prop.OperationalRule == nil {
				continue
			}
			opCtx := operational_rule.OperationalRuleContext{
				Property: prop,
				Solution: ctx,
				Data:     knowledgebase.DynamicValueData{Resource: res},
			}
			canRun, err := opCtx.EvaluateIfCondition(*prop.OperationalRule)
			if err != nil || !canRun {
				continue
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
	parts := strings.Split(v.Satisfication.Classification, "#")
	if len(parts) >= 2 {
		currResources := []construct.ResourceId{v.Edge.Source}
		if v.Satisfication.AsTarget {
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
		satisfication knowledgebase.EdgePathSatisfaction) error {
		var tempGraph construct.Graph
		if !strings.Contains(satisfication.Classification, "#") {
			tempGraph, err = path_selection.BuildPathSelectionGraph(edge, kb, satisfication.Classification)
			if err != nil {
				return fmt.Errorf("error in AddPath: could not build path selection graph: %w", err)
			}
		}
		vertex := pathExpandVertex{Edge: edge, Satisfication: satisfication, TempGraph: tempGraph}
		return vs.AddDependencies(eval.Solution, &vertex)
	}

	kb := eval.Solution.KnowledgeBase()

	edge := construct.SimpleEdge{Source: source, Target: target}
	pathSatisfications, err := kb.GetPathSatisfactionsFromEdge(source, target)

	vs := make(verticesAndDeps)
	var errs error
	for _, satisfication := range pathSatisfications {
		errs = errors.Join(errs, generateAndAddVertex(vs, edge, kb, satisfication))
	}
	if len(pathSatisfications) == 0 {
		errs = errors.Join(errs, generateAndAddVertex(vs, edge, kb, knowledgebase.EdgePathSatisfaction{}))
	}
	return errors.Join(errs, eval.enqueue(vs))
}

func DeterminePathSatisfactionInputs(
	sol solution_context.SolutionContext,
	satisfaction knowledgebase.EdgePathSatisfaction,
	edge construct.ResourceEdge,
) (expansions []ExpansionInput, errs error) {
	if !strings.Contains(satisfaction.Classification, "#") {
		expansions = append(expansions, ExpansionInput{Dep: edge, Classification: satisfaction.Classification})
		return
	}

	parts := strings.Split(satisfaction.Classification, "#")

	resources := []construct.ResourceId{edge.Source.ID}
	if satisfaction.AsTarget {
		resources = []construct.ResourceId{edge.Target.ID}
	}

	for i, part := range parts {
		fieldValueResources := []construct.ResourceId{}
		if i == 0 {
			continue
		}
		for _, resId := range resources {
			r, err := sol.RawView().Vertex(resId)
			if r == nil || err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to determine path satisfaction inputs. could not find resource %s: %w",
					resId, err,
				))
				continue
			}
			val, err := r.GetProperty(part)
			if err != nil || val == nil {
				continue
			}
			if id, ok := val.(construct.ResourceId); ok {
				fieldValueResources = append(fieldValueResources, id)
			} else if rval := reflect.ValueOf(val); rval.Kind() == reflect.Slice || rval.Kind() == reflect.Array {
				for i := 0; i < rval.Len(); i++ {
					idVal := rval.Index(i).Interface()
					if id, ok := idVal.(construct.ResourceId); ok {
						fieldValueResources = append(fieldValueResources, id)
					} else {
						errs = errors.Join(errs, fmt.Errorf(
							"failed to determine path satisfaction inputs. array property %s on resource %s is not a resource id",
							part, resId,
						))
					}
				}
			} else {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to determine path satisfaction inputs. property %s on resource %s is not a resource id",
					part, resId,
				))
			}
		}
		resources = fieldValueResources
	}
	for _, resId := range resources {
		res, err := sol.RawView().Vertex(resId)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction inputs. could not find resource %s: %w",
				resId, err,
			))
			continue
		}
		e := construct.ResourceEdge{Source: res, Target: edge.Target}
		if satisfaction.AsTarget {
			e = construct.ResourceEdge{Source: edge.Source, Target: res}
		}
		exp := ExpansionInput{
			Dep:            e,
			Classification: strings.SplitN(satisfaction.Classification, "#", 2)[0],
		}
		expansions = append(expansions, exp)
	}
	return
}
