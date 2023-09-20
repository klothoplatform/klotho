package path_selection

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	Path struct {
		Nodes  []construct.ResourceId
		Weight int
	}

	// EdgeConstraint is an object defined on EdgeData which can influence the path picked when expansion occurs.
	EdgeConstraint struct {
		// NodeMustExist specifies a list of resources which must exist in the path when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustExist []construct.Resource
		// NodeMustNotExist specifies a list of resources which must not exist when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustNotExist []construct.Resource
	}

	// EdgeData is an object attached to edges in the ResourceGraph to help the knowledge base understand context when performing expansion and configuration tasks
	EdgeData struct {
		// Constraint refers to the EdgeConstraints defined during the edge expansion
		Constraint EdgeConstraint
		// Attributes is a map of attributes which can be used to store arbitrary data on the edge
		Attributes map[string]any
	}

	Graph interface {
		GetAllDownstreamResources(res construct.Resource) []construct.Resource
	}

	KB interface {
		GetFunctionality(id construct.ResourceId) construct.Functionality
		AllPaths(source construct.ResourceId, destination construct.ResourceId) ([][]*knowledgebase.ResourceTemplate, error)
		HasDirectPath(source construct.ResourceId, destination construct.ResourceId) bool
		GetEdgeTemplate(source construct.ResourceId, destination construct.ResourceId) *knowledgebase.EdgeTemplate
	}

	PathSelectionContext struct {
		KB                   KB
		Graph                Graph
		CreateResourcefromId func(id construct.ResourceId) construct.Resource
	}
)

func (ctx PathSelectionContext) SelectPath(dep graph.Edge[construct.Resource]) ([]graph.Edge[construct.Resource], error) {
	var result []graph.Edge[construct.Resource]
	edgeData, err := getEdgeData(dep)
	if err != nil {
		return result, err
	}
	// if its a direct edge and theres no constraint on what needs to exist then we should be able to just return
	if ctx.KB.HasDirectPath(dep.Source.Id(), dep.Destination.Id()) && len(edgeData.Constraint.NodeMustExist) == 0 {
		return result, nil
	}

	paths, err := ctx.determineCorrectPaths(dep, edgeData)
	if err != nil {
		zap.S().Warnf("got error when determining correct path for edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
		return result, err
	}
	if len(paths) == 0 {
		return result, fmt.Errorf("no paths found that satisfy the attributes, %s, and do not contain unnecessary hops for edge %s -> %s", edgeData.Attributes, dep.Source.Id(), dep.Destination.Id())
	}
	path := ctx.findOptimalPath(paths)
	if len(path.Nodes) == 0 {
		return result, fmt.Errorf("empty path found that satisfy the attributes, %s, and do not contain unnecessary hops for edge %s -> %s", edgeData.Attributes, dep.Source.Id(), dep.Destination.Id())
	}
	return ctx.ExpandEdge(&dep, path, edgeData)
}

// getEdgeData retrieves the edge data from the edge in the resource graph to use during expansion
func getEdgeData(dep graph.Edge[construct.Resource]) (EdgeData, error) {
	// We want to retrieve the edge data from the edge in the resource graph to use during expansion
	edgeData := EdgeData{}
	data, ok := dep.Properties.Data.(EdgeData)
	if !ok && dep.Properties.Data != nil {
		return edgeData, fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format during expansion", dep.Source.Id(), dep.Destination.Id())
	} else if dep.Properties.Data != nil {
		edgeData = data
	}
	return edgeData, nil
}

// determineCorrectPath determines the correct path to take to get from the dependency's source node to destination node, using the knowledgebase of edges
// It first finds all possible paths given the initial source and destination node. It then filters out any paths that do not satisfy the constraints of the edge
// It then filters out any paths that contain unnecessary hops to get to the destination
func (ctx PathSelectionContext) determineCorrectPaths(dep graph.Edge[construct.Resource], edgeData EdgeData) ([]Path, error) {
	paths, err := ctx.KB.AllPaths(dep.Source.Id(), dep.Destination.Id())
	if err != nil {
		return nil, err
	}
	var validPaths []Path
	var satisfyAttributeData []Path
	for _, p := range paths {

		// Check if the path satisfies all constraints on the edge
		types := []string{}
		for _, res := range p {
			types = append(types, res.QualifiedTypeName)
		}
		for _, c := range edgeData.Constraint.NodeMustExist {
			if !collectionutil.Contains(types, c.Id().QualifiedTypeName()) {
				continue
			}
		}
		for _, c := range edgeData.Constraint.NodeMustNotExist {
			if collectionutil.Contains(types, c.Id().QualifiedTypeName()) {
				continue
			}
		}

		satisfies := true
		path := Path{}
		for _, resTemplate := range p {
			path.Nodes = append(path.Nodes, resTemplate.Id())
			isSource := resTemplate.QualifiedTypeName == dep.Source.Id().QualifiedTypeName()
			isDest := resTemplate.QualifiedTypeName == dep.Destination.Id().QualifiedTypeName()
			if ctx.KB.GetFunctionality(resTemplate.Id()) != construct.Unknown {
				path.Weight += 1
			}
			for k := range edgeData.Attributes {

				// If its a direct edge we need to make sure the source contains the attributes, otherwise ignore the source and destination of the dependency in checking if the edge satisfies the attributes
				if (!isSource && !isDest) || len(p) == 2 {
					if !collectionutil.Contains(resTemplate.Classification.Is, k) {
						satisfies = false
						break
					}
				}
			}
			if !satisfies {
				break
			}
		}
		if satisfies {
			satisfyAttributeData = append(satisfyAttributeData, path)
		}
	}
	if len(satisfyAttributeData) == 0 {
		return nil, fmt.Errorf("no paths found that satisfy the attributes, %s, for edge %s -> %s", edgeData.Attributes, dep.Source.Id(), dep.Destination.Id())
	}
	for _, p := range satisfyAttributeData {
		// Ensure we arent taking unnecessary hops to get to the destination
		if !ctx.containsUnneccessaryHopsInPath(dep, p.Nodes, edgeData) {
			validPaths = append(validPaths, p)
		}
	}
	return validPaths, nil
}

// containsUnneccessaryHopsInPath determines if the path contains any unnecessary hops to get to the destination
//
// We check if the source and destination of the dependency have a functionality. If they do, we check if the functionality of the source or destination
// is the same as the functionality of the source or destination of the edge in the path. If it is then we ensure that the source or destination of the edge
// in the path is not the same as the source or destination of the dependency. If it is then we know that the edge in the path is an unnecessary hop to get to the destination
func (ctx PathSelectionContext) containsUnneccessaryHopsInPath(dep graph.Edge[construct.Resource], p []construct.ResourceId, edgeData EdgeData) bool {
	var mustExistTypes []string
	for _, res := range edgeData.Constraint.NodeMustExist {
		mustExistTypes = append(mustExistTypes, res.Id().QualifiedTypeName())
	}

	// Track the functionality we find in the path to make sure we dont duplicate resource functions
	foundFunc := map[construct.Functionality]bool{}

	// Here we check if the edge or destination functionality exist within the path in another resource. If they do, we know that the path contains unnecessary hops.
	for i, res := range p {

		// We know that we can skip over the initial source and dest since those are the original edges passed in
		if i == 0 || i == len(p)-1 {
			continue
		}

		resFunctionality := ctx.KB.GetFunctionality(res)

		srcFunctionality := ctx.KB.GetFunctionality(dep.Source.Id())

		dstFunctionality := ctx.KB.GetFunctionality(dep.Destination.Id())

		// If one of the resources in the path has duplicate functionality as the source or destination of the dependency, we know that the path contains unnecessary hops, so check to see if it exists due to a constraint
		if res.QualifiedTypeName() == dep.Destination.Id().QualifiedTypeName() || res.QualifiedTypeName() == dep.Source.Id().QualifiedTypeName() {
			if !collectionutil.Contains(mustExistTypes, res.QualifiedTypeName()) {
				return true
			}
			// Continue here because we know the functionalities will conflict and since it must be an included node, we dont want to consider it unnecessary
			continue
		}

		if dstFunctionality != construct.Unknown && dstFunctionality == resFunctionality {
			return true
		}
		if srcFunctionality != construct.Unknown && srcFunctionality == resFunctionality {
			return true
		}

		// Now we will look to see if there are duplicate functionality in resources within the edge, if there are we will say it contains unnecessary hops. We will verify first that those duplicates dont exist because of a constraint
		if resFunctionality != construct.Unknown {
			if foundFunc[resFunctionality] {
				return true
			}
			foundFunc[resFunctionality] = true
		}
	}
	return false
}

// findOptimal path looks for the lowest weight and then shortest path of that weight.
// If there are multiple paths, the keys are sorted to be deterministic in which path is chosen
func (ctx PathSelectionContext) findOptimalPath(paths []Path) Path {
	var validPath Path

	var sameLengthPaths []Path
	// Get the shortest route that satisfied constraints
	for _, path := range paths {
		if len(validPath.Nodes) == 0 {
			validPath = path
		} else if path.Weight < validPath.Weight {
			validPath = path
			sameLengthPaths = []Path{}
		} else if path.Weight == validPath.Weight {
			if len(path.Nodes) < len(validPath.Nodes) {
				validPath = path
				sameLengthPaths = []Path{}
			} else if len(path.Nodes) == len(validPath.Nodes) {
				sameLengthPaths = append(sameLengthPaths, path, validPath)
			}
		}
	}
	// If there are multiple paths with the same length we are going to generate a string for each and sort them so we can be deterministic in which one we choose
	if len(sameLengthPaths) > 0 {
		pathStrings := map[string]Path{}
		for _, p := range sameLengthPaths {
			pString := ""
			for _, r := range p.Nodes {
				pString += fmt.Sprintf("%s ->", r)
			}
			pathStrings[pString] = p
		}
		keys := make([]string, 0, len(pathStrings))
		for k := range pathStrings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return pathStrings[keys[0]]
	}
	return validPath
}

// ExpandEdges performs calculations to determine the proper path to be inserted into the ResourceGraph.
//
// The workflow of the edge expansion is as follows:
//   - Find shortest path given the constraints on the edge
//   - Iterate through each edge in path creating the resource if necessary
func (ctx PathSelectionContext) ExpandEdge(dep *graph.Edge[construct.Resource], validPath Path, edgeData EdgeData) ([]graph.Edge[construct.Resource], error) {

	var result []graph.Edge[construct.Resource]
	// It does not matter what order we go in as each edge should be expanded independently. They can still reuse resources since the create methods should be idempotent if resources are the same.
	zap.S().Debugf("Expanding Edge for %s -> %s", dep.Source.Id(), dep.Destination.Id())

	name := fmt.Sprintf("%s_%s", dep.Source.Id().Name, dep.Destination.Id().Name)

	var previousNode construct.Resource
	var edgeTemplate *knowledgebase.EdgeTemplate

PATH:
	for i, node := range validPath.Nodes {
		if i == 0 {
			previousNode = dep.Source
			continue
		} else if i == len(validPath.Nodes)-1 {
			result = append(result, graph.Edge[construct.Resource]{
				Source:      previousNode,
				Destination: dep.Destination,
			})
			break PATH
		}

		edgeTemplate = ctx.KB.GetEdgeTemplate(previousNode.Id(), node)

		// Create a new interface of the destination nodes type if it does not exist
		destNode := ctx.CreateResourcefromId(node)
		reflect.ValueOf(destNode).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s_%s", destNode.Id().Type, name)))
		// Determine if the destination node is the same type as what is specified in the constraints as must exist
		for _, mustExistRes := range edgeData.Constraint.NodeMustExist {
			if mustExistRes.Id().Type == destNode.Id().Type && mustExistRes.Id().Provider == destNode.Id().Provider && mustExistRes.Id().Namespace == destNode.Id().Namespace {
				destNode = mustExistRes
			}
		}

		// If the edge specifies that it can reuse upstream or downstream resources, we want to find the first resource which satisfies the reuse criteria and add that as the dependency.
		// If there is no resource that satisfies the reuse criteria, we want to add the original direct dependency
		switch edgeTemplate.Reuse {
		case knowledgebase.ReuseUpstream:
			upstreamResources := ctx.Graph.GetAllDownstreamResources(dep.Source)
			for _, res := range upstreamResources {
				if previousNode.Id().Type == res.Id().Type {
					result = append(result, graph.Edge[construct.Resource]{
						Source:      res,
						Destination: destNode,
					})
					break PATH
				}
			}
		case knowledgebase.ReuseDownstream:
			upstreamResources := ctx.Graph.GetAllDownstreamResources(dep.Destination)
			for _, res := range upstreamResources {
				if destNode.Id().Type == res.Id().Type {
					result = append(result, graph.Edge[construct.Resource]{
						Source:      previousNode,
						Destination: res,
					})
					break PATH
				}
			}
		}
		result = append(result, graph.Edge[construct.Resource]{
			Source:      previousNode,
			Destination: destNode,
		})
		previousNode = destNode
	}

	return result, nil
}
