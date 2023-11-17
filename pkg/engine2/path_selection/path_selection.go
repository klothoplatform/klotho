package path_selection

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

// PHANTOM_PREFIX deliberately uses an invalid character so if it leaks into an actualy input/output, it will
// fail to parse.
const PHANTOM_PREFIX = "phantom$"
const GLUE_WEIGHT = 100
const FUNCTIONAL_WEIGHT = 100000

func BuildPathSelectionGraph(
	dep construct.SimpleEdge,
	kb knowledgebase.TemplateKB,
	classification string) (construct.Graph, error) {
	tempGraph := construct.NewAcyclicGraph(graph.Weighted())

	// Check to see if there is a direct edge which satisfies the classification and if so short circuit in building the temp graph
	if et := kb.GetEdgeTemplate(dep.Source, dep.Target); et != nil && dep.Source.Namespace == dep.Target.Namespace {
		directEdgeSatisfies := false
		if collectionutil.Contains(et.Classification, classification) {
			directEdgeSatisfies = true
		}

		if !directEdgeSatisfies {
			srcRt, err := kb.GetResourceTemplate(dep.Source)
			if err != nil {
				return nil, err
			}
			dst, err := kb.GetResourceTemplate(dep.Source)
			if err != nil {
				return nil, err
			}
			if collectionutil.Contains(srcRt.Classification.Is, classification) || collectionutil.Contains(dst.Classification.Is, classification) {
				directEdgeSatisfies = true
			}
		}

		if directEdgeSatisfies {
			err := tempGraph.AddVertex(&construct.Resource{ID: dep.Source})
			if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
				return nil, fmt.Errorf("failed to add source vertex to path selection graph for %s: %w", dep, err)
			}
			err = tempGraph.AddVertex(&construct.Resource{ID: dep.Target})
			if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
				return nil, fmt.Errorf("failed to add target vertex to path selection graph for %s: %w", dep, err)
			}
			err = tempGraph.AddEdge(dep.Source, dep.Target, graph.EdgeWeight(calculateEdgeWeight(dep, dep.Source, dep.Target, 0, 0, classification, kb)))
			if err != nil {
				return nil, err
			}
			return tempGraph, nil
		}
	}

	paths, err := kb.AllPaths(dep.Source, dep.Target)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get all paths between resources while building path selection graph for %s: %w",
			dep, err,
		)
	}
	zap.S().Debugf("Found %d paths %s", len(paths), dep)
	err = tempGraph.AddVertex(&construct.Resource{ID: dep.Source})
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return nil, fmt.Errorf("failed to add source vertex to path selection graph for %s: %w", dep, err)
	}
	err = tempGraph.AddVertex(&construct.Resource{ID: dep.Target})
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return nil, fmt.Errorf("failed to add target vertex to path selection graph for %s: %w", dep, err)
	}
	for _, path := range paths {
		resourcePath := []construct.ResourceId{}
		for _, res := range path {
			resourcePath = append(resourcePath, res.Id())
		}
		if !PathSatisfiesClassification(kb, resourcePath, classification) {
			continue
		}
		var prevRes construct.ResourceId
		for i, res := range path {
			id := res.Id()
			id.Name = fmt.Sprintf("%s%s", PHANTOM_PREFIX, generateStringSuffix(5))
			if i == 0 {
				id = dep.Source
			} else if i == len(path)-1 {
				id = dep.Target
			}
			resource := &construct.Resource{ID: id}
			err = tempGraph.AddVertex(resource)
			if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
				return nil, err
			}
			if !prevRes.IsZero() {
				edgeTemplate := kb.GetEdgeTemplate(prevRes, id)
				if edgeTemplate != nil && !edgeTemplate.DirectEdgeOnly {
					err := tempGraph.AddEdge(prevRes, id, graph.EdgeWeight(calculateEdgeWeight(dep, prevRes, id, 0, 0, classification, kb)))
					if err != nil {
						return nil, err
					}
				}
			}
			prevRes = id
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
	classification string,
	kb knowledgebase.TemplateKB,
) int {
	if divideSourceBy == 0 {
		divideSourceBy = 1
	}
	if divideTargetBy == 0 {
		divideTargetBy = 1
	}

	// check to see if the resources match the classification being solved and account for their weights accordingly
	sourceTemplate, err := kb.GetResourceTemplate(source)
	if err == nil || sourceTemplate != nil {
		if collectionutil.Contains(sourceTemplate.Classification.Is, classification) {
			divideSourceBy += 10
		}
	}
	targetTemplate, err := kb.GetResourceTemplate(target)
	if err == nil || targetTemplate != nil {
		if collectionutil.Contains(targetTemplate.Classification.Is, classification) {
			divideTargetBy += 10
		}
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
		return int(float32(weight) * et.EdgeWeightMultiplier)
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
