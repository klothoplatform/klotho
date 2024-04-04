package operational_eval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

//go:generate mockgen -source=./vertex_path_expand.go --destination=../operational_eval/vertex_path_expand_mock_test.go --package=operational_eval

type (
	pathExpandVertex struct {
		// ExpandEdge is the overall edge that is being expanded
		ExpandEdge construct.SimpleEdge
		// SatisfactionEdge is the specific edge that was being expanded when the error occurred
		SatisfactionEdge construct.SimpleEdge

		TempGraph     construct.Graph
		Satisfication knowledgebase.EdgePathSatisfaction
	}

	expansionRunner interface {
		getExpansionsToRun(v *pathExpandVertex) ([]path_selection.ExpansionInput, error)
		handleResultProperties(v *pathExpandVertex, result path_selection.ExpansionResult) error
		addSubExpansion(result path_selection.ExpansionResult, expansion path_selection.ExpansionInput, v *pathExpandVertex) error
		addResourcesAndEdges(result path_selection.ExpansionResult, expansion path_selection.ExpansionInput, v *pathExpandVertex) error
		consumeExpansionProperties(expansion path_selection.ExpansionInput) error
	}

	pathExpandVertexRunner struct {
		Eval *Evaluator
	}
)

func (v *pathExpandVertex) Key() Key {
	return Key{PathSatisfication: v.Satisfication, Edge: v.SatisfactionEdge}
}

func (v *pathExpandVertex) Evaluate(eval *Evaluator) error {
	// if both the source and target are imported resources we can skip the evaluation since its just for context
	// we will ensure the edge remains
	sourceRes, err := eval.Solution.RawView().Vertex(v.SatisfactionEdge.Source)
	if err != nil {
		return fmt.Errorf("could not find source resource %s: %w", v.SatisfactionEdge.Source, err)
	}
	targetRes, err := eval.Solution.RawView().Vertex(v.SatisfactionEdge.Target)
	if err != nil {
		return fmt.Errorf("could not find target resource %s: %w", v.SatisfactionEdge.Target, err)
	}
	if sourceRes.Imported && targetRes.Imported {
		return eval.Solution.RawView().AddEdge(v.SatisfactionEdge.Source, v.SatisfactionEdge.Target)
	}

	runner := &pathExpandVertexRunner{Eval: eval}
	edgeExpander := &path_selection.EdgeExpand{Ctx: eval.Solution}
	return v.runEvaluation(eval, runner, edgeExpander)
}

func (v *pathExpandVertex) runEvaluation(eval *Evaluator, runner expansionRunner, edgeExpander path_selection.EdgeExpander) error {
	var errs error
	expansions, err := runner.getExpansionsToRun(v)
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("could not get expansions to run: %w", err))
	}
	log := eval.Log().Named("path_expand")
	if len(expansions) > 1 && log.Desugar().Core().Enabled(zap.DebugLevel) {
		log.Debugf("Expansion %s subexpansions:", v.SatisfactionEdge)
		for _, expansion := range expansions {
			log.Debugf(" %s -> %s", expansion.SatisfactionEdge.Source.ID, expansion.SatisfactionEdge.Target.ID)
		}
	}

	createExpansionErr := func(err error) error {
		return fmt.Errorf("could not run expansion %s -> %s <%s>: %w",
			v.SatisfactionEdge.Source, v.SatisfactionEdge.Target, v.Satisfication.Classification, err,
		)
	}

	for _, expansion := range expansions {
		result, err := edgeExpander.ExpandEdge(expansion)
		if err != nil {
			errs = errors.Join(errs, createExpansionErr(err))
			continue
		}

		resultStr, err := expansionResultString(result.Graph, expansion.SatisfactionEdge)
		if err != nil {
			errs = errors.Join(errs, createExpansionErr(err))
			continue
		}
		if v.Satisfication.Classification != "" {
			log.Infof("Satisfied %s for %s through %s", v.Satisfication.Classification, v.SatisfactionEdge, resultStr)
		} else {
			log.Infof("Satisfied %s -> %s through %s", v.SatisfactionEdge.Source, v.SatisfactionEdge.Target, resultStr)
		}

		if err := runner.addResourcesAndEdges(result, expansion, v); err != nil {
			errs = errors.Join(errs, createExpansionErr(err))
			continue
		}

		if err := runner.addSubExpansion(result, expansion, v); err != nil {
			errs = errors.Join(errs, createExpansionErr(err))
			continue
		}

		if err := runner.consumeExpansionProperties(expansion); err != nil {
			errs = errors.Join(errs, createExpansionErr(err))
			continue
		}

		// do this after weve added all resources and edges to the sol ctx so that we replace the ids properly
		if err := runner.handleResultProperties(v, result); err != nil {
			errs = errors.Join(errs, createExpansionErr(err))
			continue
		}
	}
	return errs
}

func (v *pathExpandVertex) UpdateFrom(other Vertex) {
	otherVertex := other.(*pathExpandVertex)
	v.TempGraph = otherVertex.TempGraph
}

// getDepsForPropertyRef takes a property reference and recurses down until the property is not filled in on the resource
// When we reach resources with missing property references, we know they are the property vertex keys we must depend on
func getDepsForPropertyRef(
	sol solution_context.SolutionContext,
	res construct.ResourceId,
	propertyRef string,
) set.Set[Key] {
	if propertyRef == "" {
		return nil
	}
	keys := make(set.Set[Key])
	cfgCtx := solution_context.DynamicCtx(sol)
	currResources := []construct.ResourceId{res}
	parts := strings.Split(propertyRef, "#")
	for _, part := range parts {
		var nextResources []construct.ResourceId
		for _, currResource := range currResources {
			keys.Add(Key{Ref: construct.PropertyRef{Resource: currResource, Property: part}})
			val, err := cfgCtx.FieldValue(part, currResource)
			if err != nil {
				// The field hasn't resolved yet. Skip it for now, future calls to dependencies will pick it up.
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

func getDepsForScripts(
	v *pathExpandVertex,
	propCtx dependencyCapturer,
) error {
	var errs error
	if v.Satisfication.Source.Script != "" {
		srcIds := construct.ResourceList{}
		err := propCtx.ExecuteDecode(v.Satisfication.Source.Script,
			knowledgebase.DynamicValueData{Resource: v.SatisfactionEdge.Source}, &srcIds)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine dependencies for path satisfaction source inputs. could not run script for %s: %w",
				v.SatisfactionEdge.Source, err,
			))
		}
	}
	if v.Satisfication.Target.Script != "" {
		targetIds := construct.ResourceList{}
		err := propCtx.ExecuteDecode(v.Satisfication.Target.Script,
			knowledgebase.DynamicValueData{Resource: v.SatisfactionEdge.Target}, &targetIds)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine dependencies for path satisfaction target inputs. could not run script for %s: %w",
				v.SatisfactionEdge.Target, err,
			))
		}
	}
	return errs
}

func (v *pathExpandVertex) Dependencies(eval *Evaluator, propCtx dependencyCapturer) error {
	changes := propCtx.GetChanges()

	srcKey := v.Key()

	changes.addEdges(srcKey, getDepsForPropertyRef(eval.Solution, v.SatisfactionEdge.Source, v.Satisfication.Source.PropertyReference))
	changes.addEdges(srcKey, getDepsForPropertyRef(eval.Solution, v.SatisfactionEdge.Target, v.Satisfication.Target.PropertyReference))

	return getDepsForScripts(v, propCtx)
}

func (runner *pathExpandVertexRunner) getExpansionsToRun(v *pathExpandVertex) ([]path_selection.ExpansionInput, error) {
	eval := runner.Eval
	var errs error
	sourceRes, err := eval.Solution.RawView().Vertex(v.SatisfactionEdge.Source)
	if err != nil {
		return nil, fmt.Errorf("could not find source resource %s: %w", v.SatisfactionEdge.Source, err)
	}
	targetRes, err := eval.Solution.RawView().Vertex(v.SatisfactionEdge.Target)
	if err != nil {
		return nil, fmt.Errorf("could not find target resource %s: %w", v.SatisfactionEdge.Target, err)
	}
	edge := construct.ResourceEdge{Source: sourceRes, Target: targetRes}
	expansions, err := path_selection.DeterminePathSatisfactionInputs(eval.Solution, v.Satisfication, edge)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	requireFullBuild := sourceRes.Imported || targetRes.Imported
	result := make([]path_selection.ExpansionInput, len(expansions))
	for i, expansion := range expansions {
		input := path_selection.ExpansionInput{
			ExpandEdge:       v.ExpandEdge,
			SatisfactionEdge: expansion.SatisfactionEdge,
			Classification:   expansion.Classification,
			TempGraph:        v.TempGraph,
		}
		if expansion.SatisfactionEdge.Source != edge.Source || expansion.SatisfactionEdge.Target != edge.Target {
			simple := construct.SimpleEdge{Source: expansion.SatisfactionEdge.Source.ID, Target: expansion.SatisfactionEdge.Target.ID}
			tempGraph, err := path_selection.BuildPathSelectionGraph(
				simple, eval.Solution.KnowledgeBase(),
				expansion.Classification, !requireFullBuild,
			)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("error getting expansions to run. could not build path selection graph: %w", err))
				continue
			}
			input.TempGraph = tempGraph
		}
		result[i] = input
	}
	return result, errs
}

func (runner *pathExpandVertexRunner) addResourcesAndEdges(
	result path_selection.ExpansionResult,
	expansion path_selection.ExpansionInput,
	v *pathExpandVertex,
) error {
	eval := runner.Eval
	adj, err := result.Graph.AdjacencyMap()
	if err != nil {
		return err
	}
	if len(adj) > 2 {
		_, err := eval.Solution.OperationalView().Edge(v.SatisfactionEdge.Source, v.SatisfactionEdge.Target)
		if err == nil {
			if err := eval.Solution.OperationalView().RemoveEdge(v.SatisfactionEdge.Source, v.SatisfactionEdge.Target); err != nil {
				return err
			}
		} else if !errors.Is(err, graph.ErrEdgeNotFound) {
			return err
		}
	} else if len(adj) == 2 {
		err = eval.Solution.RawView().AddEdge(expansion.SatisfactionEdge.Source.ID, expansion.SatisfactionEdge.Target.ID)
		if err != nil {
			return err
		}
		return eval.Solution.OperationalView().MakeEdgesOperational([]construct.Edge{
			{Source: expansion.SatisfactionEdge.Source.ID, Target: expansion.SatisfactionEdge.Target.ID},
		})
	}

	// Once the path is selected & expanded, first add all the resources to the graph
	var errs error
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

	// After all the resources, then add all the dependencies
	edges := []construct.Edge{}
	for _, edgeMap := range adj {
		for _, edge := range edgeMap {
			err := eval.Solution.OperationalView().AddEdge(edge.Source, edge.Target)
			if err != nil {
				errs = errors.Join(errs, err)
			}
			edges = append(edges, edge)
		}
	}
	if errs != nil {
		return errs
	}
	if err := eval.AddResources(resources...); err != nil {
		return err
	}
	return eval.AddEdges(edges...)
}

func (runner *pathExpandVertexRunner) addSubExpansion(
	result path_selection.ExpansionResult,
	expansion path_selection.ExpansionInput,
	v *pathExpandVertex,
) error {
	// add sub expansions returned from the result, only for the classification of this expansion
	eval := runner.Eval
	changes := newChanges()
	for _, subExpand := range result.Edges {
		pathSatisfications, err := eval.Solution.KnowledgeBase().GetPathSatisfactionsFromEdge(subExpand.Source, subExpand.Target)
		if err != nil {
			return fmt.Errorf("could not get path satisfications for sub expansion %s -> %s: %w",
				subExpand.Source, subExpand.Target, err)
		}

		for _, satisfication := range pathSatisfications {
			if satisfication.Classification == v.Satisfication.Classification {
				// we cannot evaluate these vertices immediately because we are unsure if their dependencies have settled
				changes.addNode(&pathExpandVertex{
					SatisfactionEdge: construct.SimpleEdge{Source: subExpand.Source, Target: subExpand.Target},
					TempGraph:        expansion.TempGraph,
					Satisfication:    satisfication,
				})
			}
		}
	}
	return eval.enqueue(changes)
}

func (runner *pathExpandVertexRunner) consumeExpansionProperties(expansion path_selection.ExpansionInput) error {
	delays, err := knowledgebase.ConsumeFromResource(
		expansion.SatisfactionEdge.Source,
		expansion.SatisfactionEdge.Target,
		solution_context.DynamicCtx(runner.Eval.Solution),
	)
	if err != nil {
		return err
	}
	// we add constrains for the delayed consumption here since their property has not yet been evaluated
	c := runner.Eval.Solution.Constraints()
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

// handleProperties
func (runner *pathExpandVertexRunner) handleResultProperties(
	v *pathExpandVertex,
	result path_selection.ExpansionResult,
) error {
	eval := runner.Eval
	adj, err := result.Graph.AdjacencyMap()
	if err != nil {
		return err
	}
	pred, err := result.Graph.PredecessorMap()
	if err != nil {
		return err
	}

	handleResultProperties := func(
		res *construct.Resource,
		rt *knowledgebase.ResourceTemplate,
		resources map[construct.ResourceId]graph.Edge[construct.ResourceId],
		Direction knowledgebase.Direction,
	) error {
		var errs error
		for target := range resources {
			targetRes, err := result.Graph.Vertex(target)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			errs = errors.Join(errs, rt.LoopProperties(res, func(prop knowledgebase.Property) error {
				opRuleCtx := operational_rule.OperationalRuleContext{
					Solution: eval.Solution,
					Property: prop,
					Data:     knowledgebase.DynamicValueData{Resource: res.ID},
				}
				details := prop.Details()
				if details.OperationalRule == nil || len(details.OperationalRule.Step.Resources) == 0 {
					return nil
				}
				step := details.OperationalRule.Step
				for _, selector := range step.Resources {
					if step.Direction == Direction {
						canUse, err := selector.CanUse(
							solution_context.DynamicCtx(eval.Solution),
							knowledgebase.DynamicValueData{Resource: res.ID},
							targetRes,
						)
						if canUse && err == nil && !res.Imported {
							err = opRuleCtx.SetField(res, targetRes, step)
							if err != nil {
								errs = errors.Join(errs, err)
							}
						}
					}
				}
				return nil
			}))
		}
		return errs
	}

	var errs error
	for id, downstreams := range adj {
		oldId := id
		rt, err := eval.Solution.KnowledgeBase().GetResourceTemplate(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		res, err := eval.Solution.RawView().Vertex(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		errs = errors.Join(errs, handleResultProperties(res, rt, downstreams, knowledgebase.DirectionDownstream))
		errs = errors.Join(errs, handleResultProperties(res, rt, pred[id], knowledgebase.DirectionUpstream))

		if oldId != res.ID {
			errs = errors.Join(errs, eval.UpdateId(oldId, res.ID))
		}
	}
	return errs
}

func expansionResultString(result construct.Graph, dep construct.ResourceEdge) (string, error) {
	sb := new(strings.Builder)
	handled := make(set.Set[construct.SimpleEdge])

	path, err := graph.ShortestPathStable(result, dep.Source.ID, dep.Target.ID, construct.ResourceIdLess)
	if err != nil {
		return "", fmt.Errorf("expansion result does not contain path from %s to %s: %w", dep.Source.ID, dep.Target.ID, err)
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
