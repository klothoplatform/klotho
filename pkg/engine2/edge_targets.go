package engine2

import (
	"os"
	"slices"
	"sync"
	"time"

	"github.com/alitto/pond"
	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_eval"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type (
	GetValidEdgeTargetsConfig struct {
		Resources struct {
			Sources []construct.ResourceId
			Targets []construct.ResourceId
		}
		ResourceTypes struct {
			Sources []construct.ResourceId
			Targets []construct.ResourceId
		}
		Tags []Tag
	}
	GetPossibleEdgesContext struct {
		InputGraph []byte
		GetValidEdgeTargetsConfig
	}
)

// EdgeCanBeExpanded returns true if there is a set of kb paths between the source and target
// that satisfies path satisfaction classifications for both the source and target.
//
// This is used to determine (on a best-effort basis) if an edge can be expanded
// without fully solving the graph (which is expensive).
func (e *Engine) EdgeCanBeExpanded(ctx *solutionContext, source construct.ResourceId, target construct.ResourceId) (bool, error) {

	if source.Matches(target) {
		return false, nil
	}

	sourceResource, err := ctx.OperationalView().Vertex(source)
	if err != nil {
		return false, err
	}
	targetResource, err := ctx.OperationalView().Vertex(target)
	if err != nil {
		return false, err
	}

	eval := operational_eval.NewEvaluator(ctx)
	err = eval.AddEdges(construct.Edge{
		Source: source,
		Target: target,
	})
	if err != nil {
		return false, err
	}

	satisfactions, err := e.Kb.GetPathSatisfactionsFromEdge(source, target)
	if err != nil {
		return false, err
	}

	for _, satisfaction := range satisfactions {
		classification := satisfaction.Classification
		if classification == "" {
			continue
		}

		tempGraph, err := path_selection.BuildPathSelectionGraph(
			construct.SimpleEdge{
				Source: source,
				Target: target,
			}, ctx.KnowledgeBase(), classification)
		if err != nil {
			return false, err
		}

		_, err = path_selection.ExpandEdge(ctx, path_selection.ExpansionInput{
			Dep: construct.ResourceEdge{
				Source: sourceResource,
				Target: targetResource,
			},
			Classification: classification,
			TempGraph:      tempGraph,
		})
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func ReadGetValidEdgeTargetsConfig(path string) (GetValidEdgeTargetsConfig, error) {
	var config GetValidEdgeTargetsConfig
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

/*
GetValidEdgeTargets returns a map of valid edge targets for each source resource in the supplied graph.
The returned map is keyed by the source resource's string representation.
The value for each source resource is a list of valid target resources.

Targets are considered valid if there is a set of kb paths between the source and target
that satisfies both source and target path satisfaction classifications.

A partial set of valid targets can be generated using the filter criteria in the context's config.
*/
func (e *Engine) GetValidEdgeTargets(context *GetPossibleEdgesContext) (map[string][]string, error) {
	inputGraph, err := unmarshallInputGraph(context.InputGraph)
	if err != nil {
		return nil, err
	}
	solutionCtx := NewSolutionContext(e.Kb)
	err = solutionCtx.LoadGraph(inputGraph)
	if err != nil {
		return nil, err
	}
	topologyGraph, err := e.GetViewsDag(DataflowView, solutionCtx)

	if err != nil {
		return nil, err
	}

	var sources []construct.ResourceId
	var targets []construct.ResourceId

	qualifiedTypeMatcher := func(id construct.ResourceId) func(otherType construct.ResourceId) bool {
		return func(otherType construct.ResourceId) bool {
			return otherType.QualifiedTypeName() == id.QualifiedTypeName()
		}
	}

	// filter resources based on the context
	ids, err := construct.TopologicalSort(topologyGraph)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		tag := GetResourceVizTag(e.Kb, DataflowView, id)
		if len(context.Tags) > 0 && !slices.Contains(context.Tags, tag) {
			continue
		}
		isSource := true
		isTarget := true
		if len(context.Resources.Sources) > 0 && !slices.Contains(context.Resources.Sources, id) {
			isSource = false
		}
		if len(context.Resources.Targets) > 0 && !slices.Contains(context.Resources.Targets, id) {
			isTarget = false
		}

		if len(context.ResourceTypes.Sources) > 0 && !slices.ContainsFunc(context.ResourceTypes.Sources, qualifiedTypeMatcher(id)) {
			isSource = false
		}

		if len(context.ResourceTypes.Targets) > 0 && !slices.ContainsFunc(context.ResourceTypes.Targets, qualifiedTypeMatcher(id)) {
			isTarget = false
		}

		if isSource {
			sources = append(sources, id)
		}
		if isTarget {
			targets = append(targets, id)
		}
	}

	results := make(chan *edgeValidity)

	//var detectionGroup sync.WaitGroup

	checkerPool := pond.New(5, 1000, pond.Strategy(pond.Lazy()))
	//knownTargetValidity := make(map[string]map[string]bool)
	//rwLock := &sync.RWMutex{}

	// get all valid-edge combinations for resource types in the supplied graph
	for _, s := range sources {
		for _, t := range targets {
			source := s
			target := t

			if source.Matches(target) {
				continue
			}

			if source.Namespace == target.Name || target.Namespace == source.Name {
				continue
			}

			path, err := graph.ShortestPath(topologyGraph, source, target)
			if len(path) > 0 && err == nil {
				continue
			}

			checkerPool.Submit(func() {

				// check if we already know the validity of this edge
				//sourceType := source.QualifiedTypeName()
				//targetType := target.QualifiedTypeName()
				//
				//isValid := false
				//previouslyEvaluated := false

				//rwLock.RLock()
				//if _, ok := knownTargetValidity[sourceType]; ok {
				//	if isValid, ok = knownTargetValidity[sourceType][targetType]; ok {
				//		previouslyEvaluated = true
				//	}
				//}
				//rwLock.RUnlock()

				// only evaluate the edge if we haven't already done so for the same source and target types
				//if !previouslyEvaluated {
				isValid, _ := e.EdgeCanBeExpanded(solutionCtx, source, target)
				//} else {
				//	zap.S().Debugf("Using cached result for %s -> %s: %t", source, target, isValid)
				//}
				zap.S().Debugf("valid target: %s -> %s: %t", source, target, isValid)
				results <- &edgeValidity{
					Source:  source,
					Target:  target,
					IsValid: isValid,
				}
				//if previouslyEvaluated {
				//	return
				//}

				// cache the result, so we don't have to recompute it for the same source and target types
				// performance benefit is unclear given potential lock contention between goroutines
				//rwLock.Lock()
				//if _, ok := knownTargetValidity[sourceType]; !ok {
				//	knownTargetValidity[sourceType] = make(map[string]bool)
				//}
				//knownTargetValidity[sourceType][targetType] = isValid
				//rwLock.Unlock()
			})
		}
	}

	output := make(map[string][]string, len(sources))
	var processResultsGroup sync.WaitGroup
	processResultsGroup.Add(1)
	go func() {
		defer processResultsGroup.Done()
		for result := range results {
			if result.IsValid {
				if _, ok := output[result.Source.String()]; !ok {
					output[result.Source.String()] = []string{}
				}
				output[result.Source.String()] = append(output[result.Source.String()], result.Target.String())
			}
		}
	}()
	checkerPool.StopAndWaitFor(60 * time.Second)
	close(results)
	processResultsGroup.Wait()

	return output, nil
}

func unmarshallInputGraph(input []byte) (construct.Graph, error) {
	var yamlGraph construct.YamlGraph
	err := yaml.Unmarshal(input, &yamlGraph)
	if err != nil {
		return nil, err
	}
	return yamlGraph.Graph, nil
}

type edgeValidity struct {
	Source  construct.ResourceId
	Target  construct.ResourceId
	IsValid bool
}
