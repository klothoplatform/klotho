package path_selection

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (

	// EdgeConstraint is an object defined on EdgeData which can influence the path picked when expansion occurs.
	EdgeConstraint struct {
		// NodeMustExist specifies a list of resources which must exist in the path when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustExist []construct.ResourceId
		// NodeMustNotExist specifies a list of resources which must not exist when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustNotExist []construct.ResourceId
	}

	// EdgeData is an object attached to edges in the ResourceGraph to help the knowledge base understand context when performing expansion and configuration tasks
	EdgeData struct {
		// Constraint refers to the EdgeConstraints defined during the edge expansion
		Constraint EdgeConstraint
		// Attributes is a map of attributes which can be used to store arbitrary data on the edge
		Attributes map[string]any
	}
)

func SelectPath(ctx solution_context.SolutionContext, dep construct.ResourceEdge, edgeData EdgeData) ([]construct.ResourceId, error) {
	kb := ctx.KnowledgeBase()
	// if its a direct edge and theres no constraint on what needs to exist then we should be able to just return
	// We need to make sure we check this since some direct paths may have the DirectEdgeOnly flag set to true in their edge template
	// This flag could then devalue the direct path when it is supposed to be used
	if kb.HasDirectPath(dep.Source.ID, dep.Target.ID) && len(edgeData.Constraint.NodeMustExist) == 0 {
		return []construct.ResourceId{dep.Source.ID, dep.Target.ID}, nil
	}

	tempGraph, err := buildTempGraph(dep, edgeData, kb)
	if err != nil {
		return nil, fmt.Errorf("could not build temp graph for path selection on edge %s -> %s, err: %s", dep.Source.ID, dep.Target.ID, err.Error())
	}

	path, err := graph.ShortestPath(tempGraph, dep.Source.ID, dep.Target.ID)
	if err != nil || len(path) == 0 {
		return nil, fmt.Errorf("could not find path for edge %s -> %s, err: %s", dep.Source, dep.Target, err.Error())
	}
	if containsUnneccessaryHopsInPath(dep, path, edgeData, kb) {
		return nil, fmt.Errorf("path for edge %s -> %s contains unnecessary hops", dep.Source, dep.Target)
	}

	return path, nil
}

// buildTempGraph will build a temporary graph that contains all resources and edges that are needed
// to determine the shortest path when selecting a path from the source of the dependency to the destination of the dependency
func buildTempGraph(dep construct.ResourceEdge, edgeData EdgeData, kb knowledgebase.TemplateKB) (graph.Graph[construct.ResourceId, construct.ResourceId], error) {
	tempGraph := graph.New(
		func(r construct.ResourceId) construct.ResourceId {
			return r
		},
		graph.Directed(),
		graph.Weighted(),
	)

	err := addResourcesToTempGraph(tempGraph, dep, edgeData, kb)
	if err != nil {
		return nil, err
	}

	err = addEdgesToTempGraph(tempGraph, dep, edgeData, kb)
	if err != nil {
		return nil, err
	}
	return tempGraph, nil
}

// addResourcesToTempGraph  will add all resources to the graph if they:
// 1. are the origin source or destination
// 2. Are specified as needing to exist within a constraint
// or if they meet the following conditions:
// 1. They meet the attributes specified
// 2. are not explicitly disallowed
// 3. are not the same type as the source or destination
// 4. They are not the same type as a resource in the must exist list
func addResourcesToTempGraph(tempGraph graph.Graph[construct.ResourceId, construct.ResourceId], dep construct.ResourceEdge, edgeData EdgeData, kb knowledgebase.TemplateKB) error {
	err := tempGraph.AddVertex(dep.Source.ID)
	if err != nil {
		return err
	}

	err = tempGraph.AddVertex(dep.Target.ID)
	if err != nil {
		return err
	}

	for _, mustExist := range edgeData.Constraint.NodeMustExist {
		err := tempGraph.AddVertex(mustExist)
		if err != nil {
			return err
		}
	}

	// We add nodes to the graph if they meet the following conditions:
	// 1. They meet the attributes specified
	// 2. are not explicitly disallowed
	// 3. are not the same type as the source or destination
	// 4. They are not the same type as a resource in the must exist list
RESOURCES:
	for _, res := range kb.ListResources() {

		// skip over any type which matches our source or target since they are automatically added
		if res.Id().Matches(dep.Source.ID) || res.Id().Matches(dep.Target.ID) {
			continue
		}

		// Since we have already added nodes which correspond to constraints, skip over adding those skeleton nodes again
		for _, mustExist := range edgeData.Constraint.NodeMustExist {
			if res.Id().Matches(mustExist) {
				continue RESOURCES
			}
		}

		// skip over any type which matches the must not exist list
		for _, mustNotExist := range edgeData.Constraint.NodeMustNotExist {
			if res.Id().Matches(mustNotExist) {
				continue RESOURCES
			}
		}

		for k := range edgeData.Attributes {
			// If its a direct edge we need to make sure the source contains the attributes, otherwise ignore the source and destination of the dependency in checking if the edge satisfies the attributes
			if !collectionutil.Contains(res.Classification.Is, k) {
				continue RESOURCES
			}
		}

		err := tempGraph.AddVertex(res.Id())
		if err != nil {
			return err
		}
	}
	return nil
}

// addEdgesToTempGraph will add all edges to the graph and substitue the source, destination, and Must Exist nodes where necessary.
//
// Weights are calculated as follows:
// 1. If the source or destination of the edge has a known functionality, we add 1 to the weight for each functionality
// 2. If the edge is a direct edge, we add 100 to the weight
// 3. If the source or destination of the edge is from a constraint node, we subtract 1000 from the weight
func addEdgesToTempGraph(tempGraph graph.Graph[construct.ResourceId, construct.ResourceId], dep construct.ResourceEdge,
	edgeData EdgeData, kb knowledgebase.TemplateKB) error {

	edges, err := kb.Edges()
	if err != nil {
		return err
	}
	for _, edge := range edges {

		edgeTemplate := kb.GetEdgeTemplate(edge.Source.Id(), edge.Target.Id())
		weight := 0

		if edge.Source.GetFunctionality() != knowledgebase.Unknown {
			weight++
		}
		if edge.Target.GetFunctionality() != knowledgebase.Unknown {
			weight++
		}
		if edgeTemplate.DirectEdgeOnly {
			weight += 100
		}

		srcId := edge.Source.Id()
		dstId := edge.Target.Id()
		// Now we will add edges to the graph if they have a vertex of the same type in the graph
		// We will need to check if the type of the src and dst are any of the dependency source, dest, or must exist constraints
		// and add edges for all of the above to build a complete graph
		if edge.Source.Id().Matches(dep.Source.ID) {
			srcId = dep.Source.ID
		}
		if edge.Target.Id().Matches(dep.Target.ID) {
			dstId = dep.Target.ID
		}
		for _, mustExist := range edgeData.Constraint.NodeMustExist {
			if edge.Source.Id().Matches(mustExist) {
				err := tempGraph.AddEdge(mustExist, dstId, graph.EdgeWeight(weight-1000))
				if err != nil {
					return err
				}
			}
			if edge.Target.Id().Matches(mustExist) {
				err := tempGraph.AddEdge(srcId, mustExist, graph.EdgeWeight(weight-1000))
				if err != nil {
					return err
				}
			}
		}

		err := tempGraph.AddEdge(srcId, dstId, graph.EdgeWeight(weight))
		// If the vertex isnt found its because the edge is not relevant to the path selection
		// the scenarios could be:
		// 1. the src or dst id are of a skeleton type when we dont add the resource to the graph
		//  1. a. this is because the node is added from a node must exist constraint
		//  1. b. or because the node is not relevant to the path (doesnt satisfy attributes, etc)
		// 2. The edge is not in a relevant direction
		//  2. a. the edge is an incoming edge to the src of the edge
		//  2. b. the edge is an outgoing edge from the dst of the edge
		if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
			return err
		}

	}
	return nil
}

// containsUnneccessaryHopsInPath determines if the path contains any unnecessary hops to get to the destination
//
// We check if the source and destination of the dependency have a functionality. If they do, we check if the functionality of the source or destination
// is the same as the functionality of the source or destination of the edge in the path. If it is then we ensure that the source or destination of the edge
// in the path is not the same as the source or destination of the dependency. If it is then we know that the edge in the path is an unnecessary hop to get to the destination
func containsUnneccessaryHopsInPath(dep construct.ResourceEdge, p []construct.ResourceId, edgeData EdgeData, kb knowledgebase.TemplateKB) bool {
	if len(p) == 2 {
		return false
	}

	// Track the functionality we find in the path to make sure we dont duplicate resource functions
	foundFunc := map[knowledgebase.Functionality]bool{}
	srcTemplate, err := kb.GetResourceTemplate(dep.Source.ID)
	if err != nil {
		return false
	}
	dstTempalte, err := kb.GetResourceTemplate(dep.Target.ID)
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

		resTemplate, err := kb.GetResourceTemplate(res)
		if err != nil {
			return false
		}
		resFunctionality := resTemplate.GetFunctionality()

		if collectionutil.Contains(edgeData.Constraint.NodeMustExist, res) {
			continue
		}

		// Now we will look to see if there are duplicate functionality in resources within the edge, if there are we will say it contains unnecessary hops. We will verify first that those duplicates dont exist because of a constraint
		if resFunctionality != knowledgebase.Unknown {
			if foundFunc[resFunctionality] {
				return true
			}
			foundFunc[resFunctionality] = true
		}
	}
	return false
}
