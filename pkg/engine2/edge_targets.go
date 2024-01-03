package engine2

import (
	"github.com/alitto/pond"
	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"os"
	"slices"
	"sync"
	"time"
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

	satisfactions, err := e.Kb.GetPathSatisfactionsFromEdge(source, target)
	if err != nil {
		return false, err
	}

	sourceClassifications := make(map[construct.ResourceId]set.Set[string])
	targetClassifications := make(map[construct.ResourceId]set.Set[string])

	for _, satisfaction := range satisfactions {
		if satisfaction.Source.Classification != "" {
			if satisfaction.Source.PropertyReference == "" {
				if _, ok := sourceClassifications[source]; !ok {
					sourceClassifications[source] = make(set.Set[string])
				}
				sourceClassifications[source].Add(satisfaction.Source.Classification)
			} else {
				refs, err := solution_context.GetResourcesFromPropertyReference(ctx, source, satisfaction.Source.PropertyReference)
				if err != nil {
					return false, err
				}
				if len(refs) < 1 {
					return false, nil
				}
				srcRef := refs[len(refs)-1]
				if _, ok := sourceClassifications[srcRef]; !ok {
					sourceClassifications[srcRef] = make(set.Set[string])
				}
				sourceClassifications[srcRef].Add(satisfaction.Source.Classification)
			}
		}
		if satisfaction.Target.Classification != "" {
			if satisfaction.Target.PropertyReference == "" {
				if _, ok := targetClassifications[target]; !ok {
					targetClassifications[target] = make(set.Set[string])
				}
				targetClassifications[target].Add(satisfaction.Target.Classification)
			} else {
				refs, err := solution_context.GetResourcesFromPropertyReference(ctx, target, satisfaction.Target.PropertyReference)
				if err != nil {
					return false, err
				}
				if len(refs) < 1 {
					return false, nil
				}
				targetRef := refs[len(refs)-1]
				if _, ok := targetClassifications[targetRef]; !ok {
					targetClassifications[targetRef] = make(set.Set[string])
				}
				targetClassifications[targetRef].Add(satisfaction.Target.Classification)
			}
		}
	}

	for src, classifications := range sourceClassifications {
		for classification := range classifications {
			edge := construct.SimpleEdge{
				Source: src,
				Target: target,
			}

			zap.S().Debugf("Checking for valid path between %s and %s for classification %s", src, target, classification)
			pathSelectionGraph, err := path_selection.BuildPathSelectionGraph(edge, e.Kb, classification)
			if err != nil {
				return false, err
			}
			paths, err := graph.AllPathsBetween(pathSelectionGraph, edge.Source, edge.Target)
			if len(paths) == 0 || err != nil {
				return false, err
			}

			hasValidPath := false
			for _, path := range paths {
				if path_selection.PathSatisfiesClassification(e.Kb, path, classification) {
					hasValidPath = true
					break
				}
			}
			if !hasValidPath {
				return false, nil
			}
		}
	}

	//for tgt, classifications := range targetClassifications {
	//	for classification := range classifications {
	//		edge := construct.SimpleEdge{
	//			Source: source,
	//			Target: tgt,
	//		}
	//
	//		zap.S().Debugf("Checking for valid path between %s and %s for classification %s", source, tgt, classification)
	//		pathSelectionGraph, err := path_selection.BuildPathSelectionGraph(edge, e.Kb, classification)
	//		if err != nil {
	//			return false, err
	//		}
	//		paths, err := graph.AllPathsBetween(pathSelectionGraph, edge.Source, edge.Target)
	//		if len(paths) == 0 || err != nil {
	//			return false, err
	//		}
	//
	//		hasValidPath := false
	//		for _, path := range paths {
	//			if path_selection.PathSatisfiesClassification(e.Kb, path, classification) {
	//				hasValidPath = true
	//				break
	//			}
	//		}
	//		if !hasValidPath {
	//			return false, nil
	//		}
	//	}
	//}

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
	err = construct.WalkGraph(topologyGraph, func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		tag := e.GetResourceVizTag(string(DataflowView), id)
		if len(context.Tags) > 0 && !slices.Contains(context.Tags, tag) {
			return nil
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

		return nil
	})
	if err != nil {
		return nil, err
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
