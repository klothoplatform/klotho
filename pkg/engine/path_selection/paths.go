package path_selection

import (
	"errors"
	"fmt"
	"slices"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
)

func GetPaths(
	sol solution.Solution,
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
					if !pathSatisfiesClassification(sol.KnowledgeBase(), path, expansion.Classification) {
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
	sol solution.Solution,
	satisfaction knowledgebase.EdgePathSatisfaction,
	edge construct.ResourceEdge,
) (expansions []ExpansionInput, errs error) {
	srcIds := construct.ResourceList{edge.Source.ID}
	targetIds := construct.ResourceList{edge.Target.ID}
	var err error
	if satisfaction.Source.PropertyReferenceChangesBoundary() {
		srcIds, err = solution.GetResourcesFromPropertyReference(sol, edge.Source.ID, satisfaction.Source.PropertyReference)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction inputs. could not find resource %s: %w",
				edge.Source.ID, err,
			))
		}
	}
	if satisfaction.Target.PropertyReferenceChangesBoundary() {
		targetIds, err = solution.GetResourcesFromPropertyReference(sol, edge.Target.ID, satisfaction.Target.PropertyReference)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction inputs. could not find resource %s: %w",
				edge.Target.ID, err,
			))

		}
	}
	if satisfaction.Source.Script != "" {
		dynamicCtx := solution.DynamicCtx(sol)
		err = dynamicCtx.ExecuteDecode(satisfaction.Source.Script,
			knowledgebase.DynamicValueData{Resource: edge.Source.ID}, &srcIds)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction source inputs. could not run script for %s: %w",
				edge.Source.ID, err,
			))
		}
	}
	if satisfaction.Target.Script != "" {
		dynamicCtx := solution.DynamicCtx(sol)
		err = dynamicCtx.ExecuteDecode(satisfaction.Target.Script,
			knowledgebase.DynamicValueData{Resource: edge.Target.ID}, &targetIds)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf(
				"failed to determine path satisfaction target inputs. could not run script for %s: %w",
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

func pathSatisfiesClassification(
	kb knowledgebase.TemplateKB,
	path []construct.ResourceId,
	classification string,
) bool {
	if containsUnneccessaryHopsInPath(path, kb) {
		return false
	}
	if classification == "" {
		return true
	}
	metClassification := false
	for i, res := range path {
		resTemplate, err := kb.GetResourceTemplate(res)
		if err != nil || slices.Contains(resTemplate.PathSatisfaction.DenyClassifications, classification) {
			return false
		}
		if slices.Contains(resTemplate.Classification.Is, classification) {
			metClassification = true
		}
		if i > 0 {
			et := kb.GetEdgeTemplate(path[i-1], res)
			if slices.Contains(et.Classification, classification) {
				metClassification = true
			}
		}
	}
	return metClassification
}

// containsUnneccessaryHopsInPath determines if the path contains any unnecessary hops to get to the destination
//
// We check if the source and destination of the dependency have a functionality. If they do, we check if the functionality of the source or destination
// is the same as the functionality of the source or destination of the edge in the path. If it is then we ensure that the source or destination of the edge
// in the path is not the same as the source or destination of the dependency. If it is then we know that the edge in the path is an unnecessary hop to get to the destination
func containsUnneccessaryHopsInPath(p []construct.ResourceId, kb knowledgebase.TemplateKB) bool {
	if len(p) == 2 {
		return false
	}
	// Here we check if the edge or destination functionality exist within the path in another resource. If they do, we know that the path contains unnecessary hops.
	for i, res := range p {

		// We know that we can skip over the initial source and dest since those are the original edges passed in
		if i == 0 || i == len(p)-1 {
			continue
		}

		resTemplate, err := kb.GetResourceTemplate(res)
		if err != nil {
			return true
		}
		resFunctionality := resTemplate.GetFunctionality()
		// Now we will look to see if there are duplicate functionality in resources within the edge, if there are we will say it contains unnecessary hops. We will verify first that those duplicates dont exist because of a constraint
		if resFunctionality != knowledgebase.Unknown {
			return true
		}
	}
	return false
}
