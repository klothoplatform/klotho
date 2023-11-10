package path_selection

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

// PHANTOM_PREFIX deliberately uses an invalid character so if it leaks into an actualy input/output, it will
// fail to parse.
const PHANTOM_PREFIX = "phantom$"
const GLUE_WEIGHT = 100
const FUNCTIONAL_WEIGHT = 10000

func BuildPathSelectionGraph(
	dep construct.SimpleEdge,
	kb knowledgebase.TemplateKB,
	classification string) (construct.Graph, error) {

	tempGraph := construct.NewAcyclicGraph(graph.Weighted())
	paths, err := kb.AllPaths(dep.Source, dep.Target)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get all paths between resources while building path selection graph for %s: %w",
			dep, err,
		)
	}
	err = tempGraph.AddVertex(&construct.Resource{ID: dep.Source})
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return nil, fmt.Errorf("failed to add source vertex to path selection graph for %s: %w", dep, err)
	}
	err = tempGraph.AddVertex(&construct.Resource{ID: dep.Target})
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return nil, fmt.Errorf("failed to add target vertex to path selection graph for %s: %w", dep, err)
	}

	phantoms := make(map[string][]construct.ResourceId)

	addEdge := func(source, target construct.ResourceId) error {
		return tempGraph.AddEdge(
			source,
			target,
			graph.EdgeWeight(calculateEdgeWeight(dep, source, target, 0, 0, kb)),
		)
	}

	for _, path := range paths {
		resourcePath := make([]construct.ResourceId, len(path))
		for i, rt := range path {
			if i == 0 {
				resourcePath[i] = dep.Source
			} else if i == len(path)-1 {
				resourcePath[i] = dep.Target
			} else {
				resourcePath[i] = rt.Id()
			}
		}
		if !PathSatisfiesClassification(kb, resourcePath, classification) {
			continue
		}
		pathResources := make([][]construct.ResourceId, len(path))
		for i, id := range resourcePath {
			if i == 0 {
				pathResources[i] = []construct.ResourceId{dep.Source}
				continue
			}
			if i == len(path)-1 {
				pathResources[i] = []construct.ResourceId{dep.Target}
				for _, prev := range pathResources[i-1] {
					err := addEdge(prev, dep.Target)
					if err != nil {
						return nil, err
					}
				}
			} else {
				needsPhantom := len(phantoms[id.QualifiedTypeName()]) == 0
				for _, p := range phantoms[id.QualifiedTypeName()] {
					usedPhantom := false
					for _, prev := range pathResources[i-1] {
						err := addEdge(prev, p)
						if errors.Is(err, graph.ErrEdgeCreatesCycle) {
							needsPhantom = true
							continue
						} else if err != nil {
							return nil, err
						}
						usedPhantom = true
					}
					if usedPhantom {
						pathResources[i] = append(pathResources[i], p)
					}
				}

				if needsPhantom {
					id.Name = fmt.Sprintf("%s%s", PHANTOM_PREFIX, generateStringSuffix(5))
					phantoms[id.QualifiedTypeName()] = append(phantoms[id.QualifiedTypeName()], id)
					pathResources[i] = append(pathResources[i], id)

					err = tempGraph.AddVertex(&construct.Resource{ID: id})
					if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
						return nil, err
					}

					for _, prev := range pathResources[i-1] {
						err = addEdge(prev, id)
						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	return tempGraph, nil
}

func PathSatisfiesClassification(
	kb knowledgebase.TemplateKB,
	path []construct.ResourceId,
	classification string,
) bool {
	if ContainsUnneccessaryHopsInPath(path, kb) {
		return false
	}
	if classification == "" {
		return true
	}
	for i, res := range path {
		resTemplate, err := kb.GetResourceTemplate(res)
		if err != nil {
			return false
		}
		if collectionutil.Contains(resTemplate.Classification.Is, classification) {
			return true
		}
		if i > 0 {
			et := kb.GetEdgeTemplate(path[i-1], res)
			if collectionutil.Contains(et.Classification, classification) {
				return true
			}
		}
		if i == len(path)-1 {
			return false
		}
	}
	return true
}

func generateStringSuffix(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}

func calculateEdgeWeight(
	dep construct.SimpleEdge,
	source, target construct.ResourceId,
	divideSourceBy, divideTargetBy int,
	kb knowledgebase.TemplateKB,
) int {
	if divideSourceBy == 0 {
		divideSourceBy = 1
	}
	if divideTargetBy == 0 {
		divideTargetBy = 1
	}
	// We start with a weight of 10 for glue and 10000 for functionality for newly created edges of "phantom" resources
	// We do so to allow for the preference of existing resources since we can multiply these weights by a decimal
	// This will achieve priority for existing resources over newly created ones
	weight := 0
	if kb.GetFunctionality(source) != knowledgebase.Unknown && !source.Matches(dep.Source) {
		weight += (FUNCTIONAL_WEIGHT / divideSourceBy)
	} else {
		weight += (GLUE_WEIGHT / divideSourceBy)
	}
	if kb.GetFunctionality(target) != knowledgebase.Unknown && !target.Matches(dep.Target) {
		weight += (FUNCTIONAL_WEIGHT / divideTargetBy)
	} else {
		weight += (GLUE_WEIGHT / divideTargetBy)
	}
	et := kb.GetEdgeTemplate(source, target)
	if et != nil && et.EdgeWeightMultiplier != 0 {
		return weight * et.EdgeWeightMultiplier
	}
	return weight
}

// ContainsUnneccessaryHopsInPath determines if the path contains any unnecessary hops to get to the destination
//
// We check if the source and destination of the dependency have a functionality. If they do, we check if the functionality of the source or destination
// is the same as the functionality of the source or destination of the edge in the path. If it is then we ensure that the source or destination of the edge
// in the path is not the same as the source or destination of the dependency. If it is then we know that the edge in the path is an unnecessary hop to get to the destination
func ContainsUnneccessaryHopsInPath(p []construct.ResourceId, kb knowledgebase.TemplateKB) bool {
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
