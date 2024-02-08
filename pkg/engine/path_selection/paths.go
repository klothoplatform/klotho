package path_selection

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

func GetPaths(
	sol solution_context.SolutionContext,
	source, target construct.ResourceId,
	pathValidityChecks func(source, target construct.ResourceId, path construct.Path) bool,
	hasPathCheck bool,
) ([]construct.Path, error) {
	var errs error
	var result []construct.Path
	pathsCache := map[construct.SimpleEdge][][]construct.ResourceId{}
	pathSatisfactions, err := sol.KnowledgeBase().GetPathSatisfactionsFromEdge(source, target)
	if err != nil {
		return result, err
	}
	sourceRes, err := sol.RawView().Vertex(source)
	if err != nil {
		return result, fmt.Errorf("has path could not find source resource %s: %w", source, err)
	}
	targetRes, err := sol.RawView().Vertex(target)
	if err != nil {
		return result, fmt.Errorf("has path could not find target resource %s: %w", target, err)
	}
	edge := construct.ResourceEdge{Source: sourceRes, Target: targetRes}
	for _, satisfaction := range pathSatisfactions {
		expansions, err := DeterminePathSatisfactionInputs(sol, satisfaction, edge)
		if err != nil {
			return result, err
		}
		for _, expansion := range expansions {
			simple := construct.SimpleEdge{Source: expansion.SatisfactionEdge.Source.ID, Target: expansion.SatisfactionEdge.Target.ID}
			paths, found := pathsCache[simple]
			if !found {
				var err error
				paths, err = graph.AllPathsBetween(sol.RawView(), expansion.SatisfactionEdge.Source.ID, expansion.SatisfactionEdge.Target.ID)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
				pathsCache[simple] = paths
			}
			if len(paths) == 0 {
				return nil, nil
			}
			// we have to track the result of each expansion because if we cant find a path for a single expansion
			// we denote that we dont have an actual path from src -> target
			var expansionResult []construct.ResourceId
			if expansion.Classification != "" {
			PATHS:
				for _, path := range paths {
					for i, res := range path {
						if i == 0 {
							continue
						}
						if et := sol.KnowledgeBase().GetEdgeTemplate(path[i-1], res); et != nil && et.DirectEdgeOnly {
							continue PATHS
						}

					}
					if !PathSatisfiesClassification(sol.KnowledgeBase(), path, expansion.Classification) {
						continue PATHS
					}
					if !pathValidityChecks(source, target, path) {
						continue PATHS
					}
					result = append(result, path)
					expansionResult = path
					if hasPathCheck {
						break
					}
				}
			} else {
				expansionResult = paths[0]
				for _, path := range paths {
					result = append(result, path)
				}
				if hasPathCheck {
					break
				}
			}
			if expansionResult == nil {
				return nil, nil
			}
		}
	}
	return result, nil
}

func DeterminePathSatisfactionInputs(
	sol solution_context.SolutionContext,
	satisfaction knowledgebase.EdgePathSatisfaction,
	edge construct.ResourceEdge,
) (expansions []ExpansionInput, errs error) {
	srcIds := []construct.ResourceId{edge.Source.ID}
	targetIds := []construct.ResourceId{edge.Target.ID}
	var err error
	if satisfaction.Source.PropertyReferenceChangesBoundary() {
		srcIds, err = solution_context.GetResourcesFromPropertyReference(sol, edge.Source.ID, satisfaction.Source.PropertyReference)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction inputs. could not find resource %s: %w",
				edge.Source.ID, err,
			))
		}
	}
	if satisfaction.Target.PropertyReferenceChangesBoundary() {
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
			exp := ExpansionInput{
				SatisfactionEdge: e,
				Classification:   satisfaction.Classification,
			}
			expansions = append(expansions, exp)
		}
	}
	return
}
