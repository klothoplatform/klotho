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

const PHANTOM_PREFIX = "phantom-"
const SERVICE_API_PROVIDER = "Service"
const GLUE_WEIGHT = 100
const FUNCTIONAL_WEIGHT = 10000

var SERVICE_API_PROVIDER_CLASSIFICATIONS = []string{"network"}

func BuildPathSelectionGraph(
	dep construct.SimpleEdge,
	kb knowledgebase.TemplateKB,
	classification *string) (construct.Graph, error) {

	tempGraph := construct.NewAcyclicGraph(graph.Weighted())

	paths, err := kb.AllPaths(dep.Source, dep.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to get all paths between resources while building path selection graph for %s -> %s: %w", dep.Source, dep.Target, err)
	}
	srcTemplate, err := kb.GetResourceTemplate(dep.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource template for %s while building path selection graph for %s -> %s: %w", dep.Target, dep.Source, dep.Target, err)
	}
	err = tempGraph.AddVertex(construct.CreateResource(dep.Source))
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return nil, fmt.Errorf("failed to add source vertex to path selection graph for %s -> %s: %w", dep.Source, dep.Target, err)
	}
	dstTemplate, err := kb.GetResourceTemplate(dep.Target)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource template for %s while building path selection graph for %s -> %s: %w", dep.Target, dep.Source, dep.Target, err)
	}
	err = tempGraph.AddVertex(construct.CreateResource(dep.Target))
	if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
		return nil, fmt.Errorf("failed to add target vertex to path selection graph for %s -> %s: %w", dep.Source, dep.Target, err)
	}

	// If the destination can be communicated through its cloud service api, then we want to have paths from that service api to our target
	// Glue weight will allow it to be chosen relatively easily, but because we favor existing resources, we should favor creating
	// vpc endpoints to satisfy the path. By multiplying glue weight * 2 we allow it to count the same as the new edges from
	// x -> cloud specific private service endpoint -> dst
	// This way if it has to create more networking resources than just the cloud specific private endpoint, we will choose
	// the public api endpoint (the assumption the compute is then not in a private network to begin with)
	var serviceApi construct.ResourceId
	if dstTemplate.HasServiceApi && (classification != nil &&
		collectionutil.Contains(SERVICE_API_PROVIDER_CLASSIFICATIONS, *classification)) {
		serviceApi = construct.ResourceId{
			Provider: SERVICE_API_PROVIDER,
			Type:     dep.Target.Type,
		}
		err := tempGraph.AddVertex(construct.CreateResource(serviceApi))
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			return nil, err
		}
		err = tempGraph.AddEdge(dep.Source, serviceApi, graph.EdgeWeight(GLUE_WEIGHT*2))
		if err != nil {
			return nil, err
		}
		if srcTemplate.GetFunctionality() == knowledgebase.Compute {
			err := tempGraph.AddEdge(serviceApi, dep.Target, graph.EdgeWeight(GLUE_WEIGHT*2))
			if err != nil {
				return nil, err
			}
		}
	}
PATHS:
	for _, path := range paths {
		if containsUnneccessaryHopsInPath(dep, path, kb) {
			continue
		}
		if classification != nil {
			for i, res := range path {
				if collectionutil.Contains(res.Classification.Is, *classification) {
					break
				}
				if i == len(path)-1 {
					continue PATHS
				}
			}
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

			// If the destination can be communicated through its cloud service api, then we want to have compute paths to that service api
			if dstTemplate.HasServiceApi && (classification != nil &&
				collectionutil.Contains(SERVICE_API_PROVIDER_CLASSIFICATIONS, *classification)) {
				if res.GetFunctionality() == knowledgebase.Compute && !res.Id().Matches(dep.Source) {
					err := tempGraph.AddEdge(id, serviceApi, graph.EdgeWeight(GLUE_WEIGHT*2))
					if err != nil {
						return nil, err
					}
				}
			}

			err := tempGraph.AddVertex(construct.CreateResource(id))
			if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
				return nil, err
			}
			if !prevRes.IsZero() {
				edgeTemplate := kb.GetEdgeTemplate(prevRes, id)
				if edgeTemplate != nil && !edgeTemplate.DirectEdgeOnly {
					err := tempGraph.AddEdge(prevRes, id, graph.EdgeWeight(calculateEdgeWeight(prevRes, id, kb)))
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

func generateStringSuffix(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)

}

func calculateEdgeWeight(
	source, target construct.ResourceId,
	kb knowledgebase.TemplateKB,
) int {
	// We start with a weight of 10 for glue and 10000 for functionality for newly created edges of "phantom" resources
	// We do so to allow for the preference of existing resources since we can multiply these weights by a decimal
	// This will achieve priority for existing resources over newly created ones
	weight := GLUE_WEIGHT
	if kb.GetFunctionality(source) != knowledgebase.Unknown {
		weight += FUNCTIONAL_WEIGHT
	}
	if kb.GetFunctionality(target) != knowledgebase.Unknown {
		weight += FUNCTIONAL_WEIGHT
	}
	return weight
}

// containsUnneccessaryHopsInPath determines if the path contains any unnecessary hops to get to the destination
//
// We check if the source and destination of the dependency have a functionality. If they do, we check if the functionality of the source or destination
// is the same as the functionality of the source or destination of the edge in the path. If it is then we ensure that the source or destination of the edge
// in the path is not the same as the source or destination of the dependency. If it is then we know that the edge in the path is an unnecessary hop to get to the destination
func containsUnneccessaryHopsInPath(dep construct.SimpleEdge, p []*knowledgebase.ResourceTemplate, kb knowledgebase.TemplateKB) bool {
	if len(p) == 2 {
		return false
	}

	// Track the functionality we find in the path to make sure we dont duplicate resource functions
	foundFunc := map[knowledgebase.Functionality]bool{}
	srcTemplate, err := kb.GetResourceTemplate(dep.Source)
	if err != nil {
		return false
	}
	dstTempalte, err := kb.GetResourceTemplate(dep.Target)
	if err != nil {
		return false
	}
	foundFunc[srcTemplate.GetFunctionality()] = true
	foundFunc[dstTempalte.GetFunctionality()] = true

	// Here we check if the edge or destination functionality exist within the path in another resource. If they do, we know that the path contains unnecessary hops.
	for i, res := range p {

		// We know that we can skip over the initial source and dest since those are the original edges passed in
		if i == 0 || i == len(p)-1 {
			continue
		}

		resFunctionality := res.GetFunctionality()
		// Now we will look to see if there are duplicate functionality in resources within the edge, if there are we will say it contains unnecessary hops. We will verify first that those duplicates dont exist because of a constraint
		if resFunctionality != knowledgebase.Unknown {
			return true
			// if foundFunc[resFunctionality] {
			// 	return true
			// }
			// foundFunc[resFunctionality] = true
		}
	}
	return false
}
