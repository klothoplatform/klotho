package engine

import (
	"context"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/solution"

	"github.com/alitto/pond"
	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/path_selection"
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
func (e *Engine) EdgeCanBeExpanded(sol *engineSolution, source construct.ResourceId, target construct.ResourceId) (result bool, cacheable bool, err error) {
	cacheable = true
	edgeExpander := path_selection.EdgeExpand{Ctx: sol}

	if source.Matches(target) {
		return false, cacheable, nil
	}

	satisfactions, err := e.Kb.GetPathSatisfactionsFromEdge(source, target)
	if err != nil {
		return false, cacheable, err
	}
	sourceSatisfactionCount := 0
	targetSatisfactionCount := 0
	for _, satisfaction := range satisfactions {
		if satisfaction.Source.Classification != "" {
			sourceSatisfactionCount++
		}
		if satisfaction.Target.Classification != "" {
			targetSatisfactionCount++
		}
	}
	if sourceSatisfactionCount == 0 || targetSatisfactionCount == 0 {
		return false, cacheable, nil
	}

	for _, satisfaction := range satisfactions {
		classification := satisfaction.Classification
		if classification == "" {
			continue
		}
		var sourceReferencedResources []construct.ResourceId
		var targetReferencedResources []construct.ResourceId

		if satisfaction.Source.PropertyReference != "" {
			cacheable = false
			sourceReferencedResources, err = solution.GetResourcesFromPropertyReference(sol, source, satisfaction.Source.PropertyReference)
			if len(sourceReferencedResources) == 0 || err != nil {
				continue // ignore satisfaction if we can't resolve the property reference
			}
		}
		if satisfaction.Target.PropertyReference != "" {
			cacheable = false
			targetReferencedResources, err = solution.GetResourcesFromPropertyReference(sol, target, satisfaction.Target.PropertyReference)
			if len(targetReferencedResources) == 0 || err != nil {
				continue // ignore satisfaction if we can't resolve the property reference
			}
		}

		tempSource := source
		if len(sourceReferencedResources) > 0 {
			tempSource = sourceReferencedResources[len(sourceReferencedResources)-1]
		}
		tempTarget := target
		if len(targetReferencedResources) > 0 {
			tempTarget = targetReferencedResources[len(targetReferencedResources)-1]
		}

		tempGraph, err := path_selection.BuildPathSelectionGraph(
			sol.Context(),
			construct.SimpleEdge{
				Source: tempSource,
				Target: tempTarget,
			}, sol.KnowledgeBase(), classification, false)
		if err != nil {
			return false, cacheable, err
		}

		tempSourceResource, err := tempGraph.Vertex(tempSource)
		if err != nil {
			continue
		}
		tempTargetResource, err := tempGraph.Vertex(tempTarget)
		if err != nil {
			continue
		}

		_, err = edgeExpander.ExpandEdge(path_selection.ExpansionInput{
			SatisfactionEdge: construct.ResourceEdge{
				Source: tempSourceResource,
				Target: tempTargetResource,
			},
			Classification: classification,
			TempGraph:      tempGraph,
		})
		if err != nil {
			return false, cacheable, err
		}
	}

	return true, cacheable, nil
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
func (e *Engine) GetValidEdgeTargets(req *GetPossibleEdgesContext) (map[string][]string, error) {
	inputGraph, err := unmarshallInputGraph(req.InputGraph)
	if err != nil {
		return nil, err
	}
	solutionCtx := NewSolution(context.TODO(), e.Kb, "", &constraints.Constraints{})
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
		if len(req.Tags) > 0 && !slices.Contains(req.Tags, tag) {
			continue
		}
		isSource := true
		isTarget := true
		if len(req.Resources.Sources) > 0 && !slices.Contains(req.Resources.Sources, id) {
			isSource = false
		}
		if len(req.Resources.Targets) > 0 && !slices.Contains(req.Resources.Targets, id) {
			isTarget = false
		}

		if len(req.ResourceTypes.Sources) > 0 && !slices.ContainsFunc(req.ResourceTypes.Sources, qualifiedTypeMatcher(id)) {
			isSource = false
		}

		if len(req.ResourceTypes.Targets) > 0 && !slices.ContainsFunc(req.ResourceTypes.Targets, qualifiedTypeMatcher(id)) {
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
	knownTargetValidity := make(map[string]map[string]bool)
	rwLock := &sync.RWMutex{}

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
				sourceType := source.QualifiedTypeName()
				targetType := target.QualifiedTypeName()
				//
				isValid := false
				previouslCached := false

				rwLock.RLock()
				if _, ok := knownTargetValidity[sourceType]; ok {
					if isValid, ok = knownTargetValidity[sourceType][targetType]; ok {
						previouslCached = true
					}
				}
				rwLock.RUnlock()
				cacheable := false
				// only evaluate the edge if we haven't already done so for the same source and target types
				if !previouslCached {
					isValid, cacheable, _ = e.EdgeCanBeExpanded(solutionCtx, source, target)
				} else {
					zap.S().Debugf("Using cached result for %s -> %s: %t", source, target, isValid)
				}
				zap.S().Debugf("valid target: %s -> %s: %t", source, target, isValid)
				results <- &edgeValidity{
					Source:  source,
					Target:  target,
					IsValid: isValid,
				}
				if previouslCached {
					return
				}

				// cache the result, so we don't have to recompute it for the same source and target types
				// performance benefit is unclear given potential lock contention between goroutines
				if cacheable {
					rwLock.Lock()
					if _, ok := knownTargetValidity[sourceType]; !ok {
						knownTargetValidity[sourceType] = make(map[string]bool)
					}
					knownTargetValidity[sourceType][targetType] = isValid
					rwLock.Unlock()
				}
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
