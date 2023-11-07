package operational_eval

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
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
		err := v.runExpansion(eval, expansion)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"could not run expansion %s -> %s <%s>: %w",
				expansion.Dep.Source.ID, expansion.Dep.Target.ID, expansion.Classification, err,
			))
		}
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
		eval.Log().Infof("Satisfied %s for %s through %s", v.Satisfication.Classification, v.Edge, resultStr)
	} else {
		eval.Log().Infof("Satisfied %s -> %s through %s", v.Edge.Source, v.Edge.Target, resultStr)
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
		input := ExpansionInput{
			Dep:            expansion.Dep,
			Classification: expansion.Classification,
			TempGraph:      v.TempGraph,
		}
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
			input.TempGraph = temp
			result = append(result, input)
		} else {
			result = append(result, input)
		}
	}
	return result, errs
}

func (v *pathExpandVertex) UpdateFrom(other Vertex) {
	otherVertex := other.(*pathExpandVertex)
	v.TempGraph = otherVertex.TempGraph
}

// addDepsFromProps checks to see if any properties in `res` match any of the `dependencies`.
// If they do, add a dependency to that property - it may set up a resource that we could reuse,
// depending on the path chosen. This is a conservative dependency, since we don't know which path
// will be chosen.
func (v *pathExpandVertex) addDepsFromProps(
	eval *Evaluator,
	changes graphChanges,
	res construct.ResourceId,
	dependencies []construct.ResourceId,
) error {
	tmpl, err := eval.Solution.KnowledgeBase().GetResourceTemplate(res)
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
			// If the property can't create resources, skip it.
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

		ref := construct.PropertyRef{Resource: res, Property: k}
		for _, dep := range dependencies {
			if dep == v.Edge.Source || dep == v.Edge.Target {
				continue
			}
			if resType.Value.Matches(dep) {
				changes.addEdge(v.Key(), Key{Ref: ref})
			}
		}
	}
	return errs
}

// addDepsFromEdge checks to see if the edge's template sets any properties via configuration rules.
// If it does, go through all the existing resources and add an incoming dependency to any that match
// the resource and property from that configuration rule.
func (v *pathExpandVertex) addDepsFromEdge(
	eval *Evaluator,
	changes graphChanges,
	edge construct.Edge,
) error {
	kb := eval.Solution.KnowledgeBase()
	tmpl := kb.GetEdgeTemplate(edge.Source, edge.Target)
	if tmpl == nil {
		return nil
	}

	allRes, err := construct.ToplogicalSort(eval.Solution.RawView())
	if err != nil {
		return err
	}

	se := construct.SimpleEdge{Source: edge.Source, Target: edge.Target}
	se.Source.Name = ""
	se.Target.Name = ""

	addDepsMatching := func(ref construct.PropertyRef) error {
		eval.Log().Debugf("Checking speculative deps for %s in %s", ref, se)
		for _, res := range allRes {
			if !ref.Resource.Matches(res) {
				continue
			}
			tmpl, err := kb.GetResourceTemplate(res)
			if err != nil {
				return err
			}
			if _, hasProp := tmpl.Properties[ref.Property]; hasProp {
				actualRef := construct.PropertyRef{
					Resource: res,
					Property: ref.Property,
				}
				changes.addEdge(Key{Ref: actualRef}, v.Key())

				eval.Log().Debugf(
					"Adding speculative dependency %s -> %s (matches %s from %s)",
					actualRef, v.Key(), ref, se,
				)
			}
		}
		return nil
	}

	dyn := solution_context.DynamicCtx(eval.Solution)

	var errs error
	for i, rule := range tmpl.OperationalRules {
		for j, cfg := range rule.ConfigurationRules {
			var err error
			data := knowledgebase.DynamicValueData{Edge: &edge}
			data.Resource, err = knowledgebase.ExecuteDecodeAsResourceId(dyn, cfg.Resource, data)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not decode resource for rule %d cfg %d: %w", i, j, err))
				continue
			}
			if data.Resource.IsZero() {
				continue
			}

			// NOTE(gg): does this need to consider `Fields`?
			field := cfg.Config.Field
			err = dyn.ExecuteDecode(field, data, &field)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not decode field for rule %d cfg %d: %w", i, j, err))
				continue
			}
			if field == "" {
				continue
			}

			ref := construct.PropertyRef{Resource: data.Resource, Property: field}
			errs = errors.Join(errs, addDepsMatching(ref))
		}
	}
	return errs
}

func (v *pathExpandVertex) Dependencies(eval *Evaluator) (graphChanges, error) {
	changes := newChanges()
	cfgCtx := solution_context.DynamicCtx(eval.Solution)
	srcKey := v.Key()

	// if we have a temp graph we can analyze the paths in it for possible dependencies on property vertices
	// if we dont, we should return what we currently have
	if v.TempGraph == nil {
		return changes, nil
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
					targetKey := Key{Ref: construct.PropertyRef{Resource: currResource, Property: part}}
					changes.addEdge(srcKey, targetKey)
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

	srcDeps, err := construct.AllDownstreamDependencies(v.TempGraph, v.Edge.Source)
	if err != nil {
		return changes, err
	}
	errs = errors.Join(errs, v.addDepsFromProps(eval, changes, v.Edge.Source, srcDeps))

	targetDeps, err := construct.AllUpstreamDependencies(v.TempGraph, v.Edge.Target)
	if err != nil {
		return changes, err
	}
	errs = errors.Join(errs, v.addDepsFromProps(eval, changes, v.Edge.Target, targetDeps))
	if errs != nil {
		return changes, errs
	}

	edges, err := v.TempGraph.Edges()
	if err != nil {
		return changes, err
	}
	for _, edge := range edges {
		errs = errors.Join(errs, v.addDepsFromEdge(eval, changes, edge))
	}

	return changes, errs
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
