package path_selection

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
)

type ExpansionInput struct {
	Dep            construct.ResourceEdge
	Classification string
	TempGraph      construct.Graph
}

type ExpansionResult struct {
	Edges []graph.Edge[construct.ResourceId]
	Graph construct.Graph
}

func ExpandEdge(
	ctx solution_context.SolutionContext,
	input ExpansionInput,
) (ExpansionResult, error) {
	tempGraph := input.TempGraph
	dep := input.Dep

	result := ExpansionResult{
		Graph: construct.NewGraph(),
	}

	defer writeGraph(input, tempGraph, result.Graph)
	var errs error
	// TODO: Revisit if we want to run on namespaces (this causes issue depending on what the namespace is)
	// A file system can be a namespace and that doesnt really fit the reason we are running this at the moment
	// errs = errors.Join(errs, runOnNamespaces(dep.Source, dep.Target, ctx, result))
	connected, err := connectThroughNamespace(dep.Source, dep.Target, ctx, result)
	if err != nil {
		errs = errors.Join(errs, err)
	}
	if !connected {
		edges, err := expandEdge(ctx, input, result.Graph)
		errs = errors.Join(errs, err)
		result.Edges = append(result.Edges, edges...)
	}
	return result, errs
}

func expandEdge(
	ctx solution_context.SolutionContext,
	input ExpansionInput,
	g construct.Graph,
) ([]graph.Edge[construct.ResourceId], error) {
	paths, err := graph.AllPathsBetween(input.TempGraph, input.Dep.Source.ID, input.Dep.Target.ID)
	if err != nil {
		return nil, err
	}
	var errs error
	// represents id to qualified type because we dont need to do that processing more than once
	for _, path := range paths {
		errs = errors.Join(errs, ExpandPath(ctx, input, path, g))
	}
	if errs != nil {
		return nil, errs
	}

	path, err := graph.ShortestPath(input.TempGraph, input.Dep.Source.ID, input.Dep.Target.ID)
	if err != nil {
		return nil, errors.Join(errs, fmt.Errorf("could not find shortest path between %s and %s: %w", input.Dep.Source.ID, input.Dep.Target.ID, err))
	}

	name := fmt.Sprintf("%s_%s", input.Dep.Source.ID.Name, input.Dep.Target.ID.Name)
	// rename phantom nodes
	result := make([]construct.ResourceId, len(path))
	for i, id := range path {
		if strings.HasPrefix(id.Name, PHANTOM_PREFIX) {
			id.Name = name
			if node, err := ctx.RawView().Vertex(id); err == nil && node != nil {
				name = fmt.Sprintf("%s_%s-2", name, id.Name)
				id.Name = name
			}
			_, props, err := input.TempGraph.VertexWithProperties(id)
			if err == nil && props.Attributes != nil {
				props.Attributes["new_name"] = name
			}
		}
		result[i] = id
	}
	resultResources, errs := addResourceListToGraph(ctx, g, result)
	if errs != nil {
		return nil, errors.Join(errs, err)
	}
	errs = errors.Join(errs, handleProperties(ctx, resultResources, g))
	edges, err := findSubExpansionsToRun(resultResources, ctx)
	return edges, errors.Join(errs, err)
}

func findSubExpansionsToRun(
	result []*construct.Resource,
	ctx solution_context.SolutionContext,
) (edges []graph.Edge[construct.ResourceId], errs error) {
	resourceTemplates := make(map[construct.ResourceId]*knowledgebase.ResourceTemplate)
	added := make(map[construct.ResourceId]map[construct.ResourceId]bool)
	getResourceTemplate := func(id construct.ResourceId) *knowledgebase.ResourceTemplate {
		rt, found := resourceTemplates[id]
		if !found {
			var err error
			rt, err = ctx.KnowledgeBase().GetResourceTemplate(id)
			if err != nil || rt == nil {
				errs = errors.Join(errs, fmt.Errorf("could not find resource template for %s: %w", id, err))
				return nil
			}
		}
		return rt
	}

	for i, res := range result {
		if i == 0 || i == len(result)-1 {
			continue
		}
		rt := getResourceTemplate(res.ID)
		if rt == nil {
			continue
		}
		if len(rt.PathSatisfaction.AsSource) != 0 {
			for j := i + 2; j < len(result); j++ {
				target := result[j]
				rt := getResourceTemplate(target.ID)
				if rt == nil {
					continue
				}
				if len(rt.PathSatisfaction.AsTarget) != 0 || j == len(result)-1 {
					if _, ok := added[res.ID]; !ok {
						added[res.ID] = make(map[construct.ResourceId]bool)
					}
					if added, ok := added[res.ID][target.ID]; !ok || !added {
						edges = append(edges, graph.Edge[construct.ResourceId]{Source: res.ID, Target: target.ID})
					}
					added[res.ID][target.ID] = true
				}
			}
		}
		// do the same logic for asTarget
		if len(rt.PathSatisfaction.AsTarget) != 0 {
			for j := i - 2; j >= 0; j-- {
				source := result[j]
				rt := getResourceTemplate(source.ID)
				if rt == nil {
					continue
				}
				if len(rt.PathSatisfaction.AsSource) != 0 || j == 0 {
					if _, ok := added[source.ID]; !ok {
						added[source.ID] = make(map[construct.ResourceId]bool)
					}
					if added, ok := added[source.ID][res.ID]; !ok || !added {
						edges = append(edges, graph.Edge[construct.ResourceId]{Source: source.ID, Target: res.ID})
					}
					added[source.ID][res.ID] = true
				}
			}
		}
	}
	return
}

func handleProperties(
	ctx solution_context.SolutionContext,
	resultResources []*construct.Resource,
	g construct.Graph,
) error {
	var errs error
	for i, res := range resultResources {
		rt, err := ctx.KnowledgeBase().GetResourceTemplate(res.ID)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		handleProp := func(prop *knowledgebase.Property) {
			oldId := res.ID
			opRuleCtx := operational_rule.OperationalRuleContext{
				Solution: ctx,
				Property: prop,
				Data:     knowledgebase.DynamicValueData{Resource: res.ID},
			}
			if prop.OperationalRule == nil {
				return
			}
			for _, step := range prop.OperationalRule.Steps {
				for _, selector := range step.Resources {
					if step.Direction == knowledgebase.DirectionDownstream && i < len(resultResources)-1 {
						downstreamRes := resultResources[i+1]
						if canUse, err := selector.CanUse(solution_context.DynamicCtx(ctx),
							knowledgebase.DynamicValueData{Resource: res.ID}, downstreamRes); canUse && err == nil {
							err = opRuleCtx.SetField(res, downstreamRes, step)
							if err != nil {
								errs = errors.Join(errs, err)
							}
						}
					} else if i > 0 {
						upstreamRes := resultResources[i-1]
						if canUse, err := selector.CanUse(solution_context.DynamicCtx(ctx),
							knowledgebase.DynamicValueData{Resource: res.ID}, upstreamRes); canUse && err == nil {
							err = opRuleCtx.SetField(res, upstreamRes, step)
							if err != nil {
								errs = errors.Join(errs, err)
							}
						}

					}
				}
			}
			if prop.Namespace && oldId.Namespace != res.ID.Namespace {
				errs = errors.Join(errs, construct.ReplaceResource(g, oldId, res))
			}
		}
		rt.LoopProperties(res, handleProp)
	}
	return errs
}

// ExpandEdge takes a given `selectedPath` and resolves it to a path of resourceIds that can be used
// for creating resources, or existing resources.
func ExpandPath(
	ctx solution_context.SolutionContext,
	input ExpansionInput,
	path []construct.ResourceId,
	resultGraph construct.Graph,
) error {
	if len(path) == 2 {
		return nil
	}

	type candidate struct {
		id             construct.ResourceId
		divideWeightBy int
	}

	var errs error

	nonBoundaryResources := path[1 : len(path)-1]

	// candidates maps the nonboundary index to the set of resources that could satisfy it
	// this is a helper to make adding all the edges to the graph easier.
	candidates := make([]map[construct.ResourceId]int, len(nonBoundaryResources))

	newResources := make(set.Set[construct.ResourceId])
	// Create new nodes for the path
	for i, node := range nonBoundaryResources {
		candidates[i] = make(map[construct.ResourceId]int)
		candidates[i][node] = 0
		newResources.Add(node)
	}
	if errs != nil {
		return errs
	}

	addCandidates := func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		matchIdx := matchesNonBoundary(id, nonBoundaryResources)
		if matchIdx < 0 {
			return nil
		}
		valid, err := checkNamespaceValidity(ctx, resource, input.Dep.Target.ID)
		if err != nil {
			return errors.Join(nerr, err)
		}
		if !valid {
			return nerr
		}

		// Calculate edge weight for candidate
		err = input.TempGraph.AddVertex(resource)
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			return errors.Join(nerr, err)
		}
		if _, ok := candidates[matchIdx][id]; !ok {
			candidates[matchIdx][id] = 0
		}
		weight, err := determineCandidateWeight(ctx, input.Dep.Source.ID, input.Dep.Target.ID, id, resultGraph)
		if err != nil {
			return errors.Join(nerr, err)
		}

		// right now we dont want validity checks to be blocking, just preference so we use them to modify the weight
		valid, err = checkCandidatesValidity(ctx, resource, path, input.Classification)
		if err != nil {
			return errors.Join(nerr, err)
		}
		if !valid {
			weight = 0
		}
		candidates[matchIdx][id] += weight
		return nerr
	}
	// We need to add candidates which exist in our current result graph so we can reuse them. We do this in case
	// we have already performed expansions to ensure the namespaces are connected, etc
	err := construct.WalkGraph(resultGraph, func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		return addCandidates(id, resource, nerr)
	})
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("error during result graph walk graph: %w", err))
	}

	// Add all other candidates which exist within the graph
	err = construct.WalkGraph(ctx.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		return addCandidates(id, resource, nerr)
	})
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("error during raw view walk graph: %w", err))
	}

	predecessors, err := ctx.DataflowGraph().PredecessorMap()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	adjacent, err := ctx.DataflowGraph().AdjacencyMap()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	// addEdge checks whether the edge should be added according to the following rules:
	// 1. If it connects two new resources, always add it
	// 2. If the edge exists, and its template specifies it is unique, only add it if it's an existing edge
	// 3. Otherwise, add it
	addEdge := func(source, target candidate) {
		weight := calculateEdgeWeight(
			construct.SimpleEdge{Source: input.Dep.Source.ID, Target: input.Dep.Target.ID},
			source.id, target.id,
			source.divideWeightBy, target.divideWeightBy,
			ctx.KnowledgeBase())

		tmpl := ctx.KnowledgeBase().GetEdgeTemplate(source.id, target.id)
		if tmpl == nil {
			errs = errors.Join(errs, fmt.Errorf("could not find edge template for %s -> %s", source.id, target.id))
			return
		}
		if tmpl.Unique.Target {
			pred := predecessors[target.id]
			for origSource := range pred {
				if tmpl.Source.Matches(origSource) && origSource != source.id {
					return
				}
			}
		}
		if tmpl.Unique.Source {
			adj := adjacent[source.id]
			for origTarget := range adj {
				if tmpl.Target.Matches(origTarget) && origTarget != target.id {
					return
				}
			}
		}

		err := input.TempGraph.AddEdge(source.id, target.id, graph.EdgeWeight(weight))
		if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) && !errors.Is(err, graph.ErrEdgeCreatesCycle) {
			errs = errors.Join(errs, err)
		}
	}

	for i, resCandidates := range candidates {
		for id, weight := range resCandidates {
			if i == 0 {
				addEdge(candidate{id: input.Dep.Source.ID}, candidate{id: id, divideWeightBy: weight})
				continue
			}

			sources := candidates[i-1]

			for source, w := range sources {
				addEdge(candidate{id: source, divideWeightBy: w}, candidate{id: id, divideWeightBy: weight})
			}

		}
	}
	if len(candidates) > 0 {
		for c, weight := range candidates[len(candidates)-1] {
			addEdge(candidate{id: c, divideWeightBy: weight}, candidate{id: input.Dep.Target.ID})
		}
	}
	if errs != nil {
		return errs
	}
	return nil
}

func runOnNamespaces(src, target *construct.Resource, ctx solution_context.SolutionContext, result ExpansionResult) error {
	if src.ID.Namespace != "" && target.ID.Namespace != "" {
		kb := ctx.KnowledgeBase()
		targetNamespaceResourceId, err := kb.GetResourcesNamespaceResource(target)
		if targetNamespaceResourceId.IsZero() {
			return fmt.Errorf("could not find namespace resource for %s", target)
		}
		if err != nil {
			return err
		}

		srcNamespaceResourceId, err := kb.GetResourcesNamespaceResource(src)
		if srcNamespaceResourceId.IsZero() {
			return fmt.Errorf("could not find namespace resource for %s", src)
		}
		if err != nil {
			return err
		}
		srcNamespaceResource, err := ctx.RawView().Vertex(srcNamespaceResourceId)
		if err != nil {
			return err
		}
		targetNamespaceResource, err := ctx.RawView().Vertex(targetNamespaceResourceId)
		if err != nil {
			return err
		}
		// if we have a namespace resource that is not the same as the target namespace resource
		tg, err := BuildPathSelectionGraph(
			construct.SimpleEdge{Source: srcNamespaceResourceId, Target: targetNamespaceResourceId},
			kb,
			"",
		)
		if err != nil {
			return fmt.Errorf("could not build path selection graph: %w", err)
		}
		// TODO: We should get all of the as source and as targets here to ensure we have all paths required
		input := ExpansionInput{
			Dep:            construct.ResourceEdge{Source: srcNamespaceResource, Target: targetNamespaceResource},
			Classification: "",
			TempGraph:      tg,
		}
		edges, err := expandEdge(ctx, input, result.Graph)
		if err != nil {
			return err
		}
		result.Edges = append(result.Edges, edges...)
	}
	return nil
}

func connectThroughNamespace(src, target *construct.Resource, ctx solution_context.SolutionContext, result ExpansionResult) (
	connected bool,
	errs error,
) {
	kb := ctx.KnowledgeBase()
	targetNamespaceResource, _ := kb.GetResourcesNamespaceResource(target)
	if targetNamespaceResource.IsZero() {
		return
	}

	downstreams, err := solution_context.Downstream(ctx, src.ID, knowledgebase.ResourceLocalLayer)
	if err != nil {
		return connected, err
	}
	for _, downId := range downstreams {
		// Right now we only check for side effects of the same type
		// We may want to check for any side effects that could be namespaced into the target namespace since that would influence
		// the source resources connection to that target namespace resource
		if downId.QualifiedTypeName() != targetNamespaceResource.QualifiedTypeName() {
			continue
		}
		down, err := ctx.RawView().Vertex(downId)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		res, _ := kb.GetResourcesNamespaceResource(down)
		if res.IsZero() {
			continue
		}
		if res == targetNamespaceResource {
			continue
		}
		// if we have a namespace resource that is not the same as the target namespace resource
		tg, err := BuildPathSelectionGraph(construct.SimpleEdge{Source: res, Target: target.ID}, kb, "")
		if err != nil {
			continue
		}
		input := ExpansionInput{
			Dep:            construct.ResourceEdge{Source: down, Target: target},
			Classification: "",
			TempGraph:      tg,
		}
		edges, err := expandEdge(ctx, input, result.Graph)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		result.Edges = append(result.Edges, edges...)
		connected = true
	}

	return
}

func addResourceListToGraph(ctx solution_context.SolutionContext, g construct.Graph, resources []construct.ResourceId) (
	[]*construct.Resource,
	error,
) {
	var result []*construct.Resource
	var errs error
	var prevRes construct.ResourceId
	for i, resource := range resources {
		r, err := ctx.RawView().Vertex(resource)
		if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
			errs = errors.Join(errs, err)
			continue
		}
		if r == nil {
			r, err = knowledgebase.CreateResource(ctx.KnowledgeBase(), resource)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		}
		err = g.AddVertex(r)
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			errs = errors.Join(errs, err)
		}
		result = append(result, r)
		if i == 0 {
			prevRes = r.ID
			continue
		}
		err = g.AddEdge(prevRes, r.ID)
		if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
			errs = errors.Join(errs, err)
		}
		prevRes = r.ID
	}
	return result, errs
}

// NOTE(gg): if for some reason the path could contain a duplicated selector
// this would just add the resource to the first match. I don't
// think this should happen for a call into [ExpandEdge], but noting it just in case.
func matchesNonBoundary(id construct.ResourceId, nonBoundaryResources []construct.ResourceId) int {
	for i, node := range nonBoundaryResources {
		typedNodeId := construct.ResourceId{Provider: node.Provider, Type: node.Type, Namespace: node.Namespace}
		if typedNodeId.Matches(id) {
			return i
		}
	}
	return -1
}
