package path_selection

import (
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func SelectPath(dep construct.Edge, edgeData EdgeData, kb knowledgebase.TemplateKB) ([]construct.ResourceId, error) {
	// if its a direct edge and theres no constraint on what needs to exist then we should be able to just return
	// We need to make sure we check this since some direct paths may have the DirectEdgeOnly flag set to true in their edge template
	// This flag could then devalue the direct path when it is supposed to be used
	if kb.HasDirectPath(dep.Source, dep.Target) && len(edgeData.Constraint.NodeMustExist) == 0 {
		return []construct.ResourceId{dep.Source, dep.Target}, nil
	}

	tempGraph, err := buildTempGraph(dep, edgeData, kb)
	if err != nil {
		return nil, fmt.Errorf("could not build temp graph for edge %s -> %s, err: %s", dep.Source, dep.Target, err.Error())
	}

	path, err := graph.ShortestPath(tempGraph, dep.Source, dep.Target)
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
func buildTempGraph(dep construct.Edge, edgeData EdgeData, kb knowledgebase.TemplateKB) (graph.Graph[construct.ResourceId, construct.ResourceId], error) {
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
func addResourcesToTempGraph(tempGraph graph.Graph[construct.ResourceId, construct.ResourceId], dep construct.Edge, edgeData EdgeData, kb knowledgebase.TemplateKB) error {
	err := tempGraph.AddVertex(dep.Source)
	if err != nil {
		return err
	}

	err = tempGraph.AddVertex(dep.Target)
	if err != nil {
		return err
	}

	for _, mustExist := range edgeData.Constraint.NodeMustExist {
		err := tempGraph.AddVertex(mustExist.ID)
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

		if res.Id().QualifiedTypeName() == dep.Source.QualifiedTypeName() || res.Id().QualifiedTypeName() == dep.Target.QualifiedTypeName() {
			continue
		}

		for _, mustExist := range edgeData.Constraint.NodeMustExist {
			if mustExist.ID.QualifiedTypeName() == res.Id().QualifiedTypeName() {
				continue RESOURCES
			}
		}

		for _, mustNotExist := range edgeData.Constraint.NodeMustNotExist {
			if mustNotExist.ID.QualifiedTypeName() == res.Id().QualifiedTypeName() {
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
func addEdgesToTempGraph(tempGraph graph.Graph[construct.ResourceId, construct.ResourceId], dep construct.Edge, edgeData EdgeData, kb knowledgebase.TemplateKB) error {

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
		if edge.Source.Id().QualifiedTypeName() == dep.Source.QualifiedTypeName() {
			srcId = dep.Source
		}
		if edge.Target.Id().QualifiedTypeName() == dep.Target.QualifiedTypeName() {
			dstId = dep.Target
		}
		for _, mustExist := range edgeData.Constraint.NodeMustExist {
			if edge.Source.Id().QualifiedTypeName() == mustExist.ID.QualifiedTypeName() {
				err := tempGraph.AddEdge(mustExist.ID, dstId, graph.EdgeWeight(weight-1000))
				if err != nil {
					return err
				}
			}
			if edge.Target.Id().QualifiedTypeName() == mustExist.ID.QualifiedTypeName() {
				err := tempGraph.AddEdge(srcId, mustExist.ID, graph.EdgeWeight(weight-1000))
				if err != nil {
					return err
				}
			}
		}

		err := tempGraph.AddEdge(srcId, dstId, graph.EdgeWeight(weight))
		if err != nil {
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
func containsUnneccessaryHopsInPath(dep construct.Edge, p []construct.ResourceId, edgeData EdgeData, kb knowledgebase.TemplateKB) bool {
	if len(p) == 2 {
		return false
	}
	var mustExistIds []construct.ResourceId
	for _, res := range edgeData.Constraint.NodeMustExist {
		mustExistIds = append(mustExistIds, res.ID)
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

		resTemplate, err := kb.GetResourceTemplate(res)
		if err != nil {
			return false
		}
		resFunctionality := resTemplate.GetFunctionality()

		if collectionutil.Contains(mustExistIds, res) {
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
