package path_selection

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/logging"
)

// PHANTOM_PREFIX deliberately uses an invalid character so if it leaks into an actual input/output, it will
// fail to parse.
const PHANTOM_PREFIX = "phantom$"
const GLUE_WEIGHT = 100
const FUNCTIONAL_WEIGHT = 100000

func BuildPathSelectionGraph(
	ctx context.Context,
	dep construct.SimpleEdge,
	kb knowledgebase.TemplateKB,
	classification string,
	ignoreDirectEdge bool,
) (construct.Graph, error) {
	log := logging.GetLogger(ctx).Sugar()
	log.Debugf("Building path selection graph for %s", dep)
	tempGraph := construct.NewAcyclicGraph(graph.Weighted())

	// Check to see if there is a direct edge which satisfies the classification and if so short circuit in building the temp graph
	et := kb.GetEdgeTemplate(dep.Source, dep.Target)
	if !ignoreDirectEdge && et != nil && dep.Source.Namespace == dep.Target.Namespace {
		directEdgeSatisfies := collectionutil.Contains(et.Classification, classification)

		if !directEdgeSatisfies {
			srcRt, err := kb.GetResourceTemplate(dep.Source)
			if err != nil {
				return nil, err
			}
			dst, err := kb.GetResourceTemplate(dep.Source)
			if err != nil {
				return nil, err
			}
			directEdgeSatisfies = collectionutil.Contains(srcRt.Classification.Is, classification) ||
				collectionutil.Contains(dst.Classification.Is, classification)
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
			err = tempGraph.AddEdge(dep.Source, dep.Target, graph.EdgeWeight(CalculateEdgeWeight(dep, dep.Source, dep.Target, 0, 0, classification, kb)))
			if err != nil {
				return nil, err
			}
			return tempGraph, nil
		}
	}

	// Panic is okay on the cast in following line since it will only happen on programming error
	kbGraph := kb.(interface{ Graph() knowledgebase.Graph }).Graph()

	err := tempGraph.AddVertex(&construct.Resource{ID: dep.Source, Properties: make(construct.Properties)})
	if err != nil {
		return nil, fmt.Errorf("failed to add source vertex to path selection graph for %s: %w", dep, err)
	}
	err = tempGraph.AddVertex(&construct.Resource{ID: dep.Target, Properties: make(construct.Properties)})
	if err != nil {
		return nil, fmt.Errorf("failed to add target vertex to path selection graph for %s: %w", dep, err)
	}

	satisfied_paths := 0
	addPath := func(path []string) error {
		var prevId construct.ResourceId
		for i, typeName := range path {
			tmpl, err := kbGraph.Vertex(typeName)
			if err != nil {
				return fmt.Errorf("failed to get template for path[%d]: %w", i, err)
			}

			var id construct.ResourceId
			switch i {
			case 0:
				prevId = dep.Source
				continue
			case len(path) - 1:
				id = dep.Target
			default:
				id, err = makePhantom(tempGraph, tmpl.Id())
				if err != nil {
					return fmt.Errorf("failed to make phantom for path[%d]: %w", i, err)
				}
				res := &construct.Resource{ID: id, Properties: make(construct.Properties)}
				if err := tempGraph.AddVertex(res); err != nil {
					return fmt.Errorf("failed to add phantom vertex for path[%d]: %w", i, err)
				}
			}

			weight := graph.EdgeWeight(CalculateEdgeWeight(dep, prevId, id, 0, 0, classification, kb))
			if err := tempGraph.AddEdge(prevId, id, weight); err != nil {
				return fmt.Errorf("failed to add edge[%d] %s -> %s: %w", i-1, prevId, id, err)
			}
			prevId = id
		}
		satisfied_paths++
		return nil
	}

	err = ClassPaths(kbGraph, dep.Source.QualifiedTypeName(), dep.Target.QualifiedTypeName(), classification, addPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find paths for %s: %w", dep, err)
	}

	log.Debugf("Found %d paths for %s :: %s", satisfied_paths, dep, classification)

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
	metClassification := false
	for i, res := range path {
		resTemplate, err := kb.GetResourceTemplate(res)
		if err != nil || slices.Contains(resTemplate.PathSatisfaction.DenyClassifications, classification) {
			return false
		}
		if collectionutil.Contains(resTemplate.Classification.Is, classification) {
			metClassification = true
		}
		if i > 0 {
			et := kb.GetEdgeTemplate(path[i-1], res)
			if collectionutil.Contains(et.Classification, classification) {
				metClassification = true
			}
		}
	}
	return metClassification
}

func makePhantom(g construct.Graph, id construct.ResourceId) (construct.ResourceId, error) {
	for suffix := 0; suffix < 1000; suffix++ {
		candidate := id
		candidate.Name = fmt.Sprintf("%s%d", PHANTOM_PREFIX, suffix)
		if _, err := g.Vertex(candidate); errors.Is(err, graph.ErrVertexNotFound) {
			return candidate, nil
		}
	}
	return id, fmt.Errorf("exhausted suffixes for creating phantom for %s", id)
}

func CalculateEdgeWeight(
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
	if knowledgebase.GetFunctionality(kb, source) != knowledgebase.Unknown && !source.Matches(dep.Source) {
		if divideSourceBy > 0 {
			weight += (FUNCTIONAL_WEIGHT / divideSourceBy)
		} else if divideSourceBy < 0 {
			weight += (FUNCTIONAL_WEIGHT * divideSourceBy * -1)
		}
	} else {
		if divideSourceBy > 0 {
			weight += (GLUE_WEIGHT / divideSourceBy)
		} else if divideSourceBy < 0 {
			weight += (GLUE_WEIGHT * divideSourceBy * -1)
		}
	}
	if knowledgebase.GetFunctionality(kb, target) != knowledgebase.Unknown && !target.Matches(dep.Target) {
		if divideTargetBy > 0 {
			weight += (FUNCTIONAL_WEIGHT / divideTargetBy)
		} else if divideTargetBy < 0 {
			weight += (FUNCTIONAL_WEIGHT * divideTargetBy * -1)
		}
	} else {
		if divideTargetBy > 0 {
			weight += (GLUE_WEIGHT / divideTargetBy)
		} else if divideTargetBy < 0 {
			weight += (GLUE_WEIGHT * divideTargetBy * -1)
		}
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
