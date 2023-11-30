package operational_eval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type (
	pathExpandVertex struct {
		Edge          construct.SimpleEdge
		TempGraph     construct.Graph
		Satisfication knowledgebase.EdgePathSatisfaction
	}
)

func (v *pathExpandVertex) Key() Key {
	return Key{PathSatisfication: v.Satisfication, Edge: v.Edge}
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

func (v *pathExpandVertex) runExpansion(eval *Evaluator, expansion path_selection.ExpansionInput) error {
	var errs error
	result, err := path_selection.ExpandEdge(eval.Solution, expansion)
	if err != nil {
		return fmt.Errorf("failed to evaluate path expand vertex. could not expand edge %s: %w", v.Edge, err)
	}

	adj, err := result.Graph.AdjacencyMap()
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
			res, err = result.Graph.Vertex(pathId)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
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

	resultStr, err := expansionResultString(result.Graph, expansion.Dep)
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
	if err := eval.AddEdges(result.Edges...); err != nil {
		return err
	}
	if err := eval.AddEdges(edges...); err != nil {
		return err
	}
	delays, err := knowledgebase.ConsumeFromResource(expansion.Dep.Source, expansion.Dep.Target, solution_context.DynamicCtx(eval.Solution))
	if err != nil {
		return err
	}
	// we add constrains for the delayed consumption here since their property has not yet been evaluated
	c := eval.Solution.Constraints()
	for _, delay := range delays {
		c.Resources = append(c.Resources, constraints.ResourceConstraint{
			Operator: constraints.AddConstraintOperator,
			Target:   delay.Resource,
			Property: delay.PropertyPath,
			Value:    delay.Value,
		})
	}
	return nil
}

func (v *pathExpandVertex) getExpansionsToRun(eval *Evaluator) ([]path_selection.ExpansionInput, error) {
	var result []path_selection.ExpansionInput
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
		input := path_selection.ExpansionInput{
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
		details := prop.Details()
		if details.OperationalRule == nil {
			// If the property can't create resources, skip it.
			continue
		}
		ready, err := operational_rule.EvaluateIfCondition(details.OperationalRule.If,
			eval.Solution, knowledgebase.DynamicValueData{Resource: res})
		if err != nil || !ready {
			continue
		}

		ref := construct.PropertyRef{Resource: res, Property: k}
		for _, dep := range dependencies {
			if dep == v.Edge.Source || dep == v.Edge.Target {
				continue
			}
			resource, err := eval.Solution.RawView().Vertex(res)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			// if this dependency could pass validation for the resources property, consider it as a dependent vertex
			if err := prop.Validate(dep, resource.Properties); err == nil {
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

			// We ignore the error because it just means that we cant resolve the resource yet
			// therefore we cant add a dependency on this invocation
			if err != nil || data.Resource.IsZero() {
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
	srcKey := v.Key()

	var errs error
	if propertyReferenceInfluencesEdge(v.Satisfication.Source) {
		keys := getDepsForPropertyRef(eval.Solution, v.Edge.Source, v.Satisfication.Source.PropertyReference)
		changes.addEdges(srcKey, keys)
	}
	if propertyReferenceInfluencesEdge(v.Satisfication.Target) {
		keys := getDepsForPropertyRef(eval.Solution, v.Edge.Target, v.Satisfication.Target.PropertyReference)
		changes.addEdges(srcKey, keys)
	}

	// if we have a temp graph we can analyze the paths in it for possible dependencies on property vertices
	// if we dont, we should return what we currently have
	// This has to be run after we analyze the refs used in path expansion to make sure the operational rules
	// dont create other resources that need to be operated on in the path expand vertex
	if v.TempGraph == nil {
		return changes, nil
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
) (expansions []path_selection.ExpansionInput, errs error) {
	srcIds := []construct.ResourceId{edge.Source.ID}
	targetIds := []construct.ResourceId{edge.Target.ID}
	var err error
	if propertyReferenceInfluencesEdge(satisfaction.Source) {
		srcIds, err = solution_context.GetResourcesFromPropertyReference(sol, edge.Source.ID, satisfaction.Source.PropertyReference)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction inputs. could not find resource %s: %w",
				edge.Source.ID, err,
			))
		}
	}
	if propertyReferenceInfluencesEdge(satisfaction.Target) {
		targetIds, err = solution_context.GetResourcesFromPropertyReference(sol, edge.Target.ID, satisfaction.Target.PropertyReference)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction inputs. could not find resource %s: %w",
				edge.Target.ID, err,
			))

		}
	}

	for _, srcId := range srcIds {
		for _, targetId := range targetIds {
			if srcId == targetId {
				continue
			}
			src, err := sol.RawView().Vertex(srcId)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to determine path satisfaction inputs. could not find resource %s: %w",
					srcId, err,
				))
				continue
			}

			target, err := sol.RawView().Vertex(targetId)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf(
					"failed to determine path satisfaction inputs. could not find resource %s: %w",
					targetId, err,
				))
				continue
			}

			e := construct.ResourceEdge{Source: src, Target: target}
			exp := path_selection.ExpansionInput{
				Dep:            e,
				Classification: satisfaction.Classification,
			}
			expansions = append(expansions, exp)
		}
	}
	return
}

// getDepsForPropertyRef takes a property reference and recurses down until the property is not filled in on the resource
// When we reach resources with missing property references, we know they are the property vertex keys we must depend on
func getDepsForPropertyRef(
	sol solution_context.SolutionContext,
	res construct.ResourceId,
	propertyRef string,
) set.Set[Key] {
	keys := make(set.Set[Key])
	cfgCtx := solution_context.DynamicCtx(sol)
	currResources := []construct.ResourceId{res}
	parts := strings.Split(propertyRef, "#")
	for _, part := range parts {
		var nextResources []construct.ResourceId
		for _, currResource := range currResources {
			val, err := cfgCtx.FieldValue(part, currResource)
			if err != nil {
				keys.Add(Key{Ref: construct.PropertyRef{Resource: currResource, Property: part}})
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
	return keys
}

func propertyReferenceInfluencesEdge(v knowledgebase.PathSatisfactionRoute) bool {
	if v.Validity != "" {
		return false
	}
	return v.PropertyReference != ""
}
