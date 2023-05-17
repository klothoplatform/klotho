package core

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type (
	// Edge defines an entry in a Knowledge base. An Edge represents a valid linking between two types of resources
	Edge struct {
		// From represents the source resource in the edge
		From reflect.Type
		// To represents the destination resource in the edge
		To reflect.Type
	}

	// EdgeDetails defines the set of characteristics and edge in the knowledge base contains. The details are used to ensure graph correctness for ResourceGraphs
	EdgeDetails struct {
		// ExpansionFunc is a function used to create the To and From resource and necessary intermediate resources, if any do not exist, to ensure the nodes are in place for correct functionality.
		ExpansionFunc EdgeExpander
		// Configure is a function used to configure the To and From resources and necessary dependent resources, to ensure the nodes will guarantee correct functionality.
		Configure EdgeConfigurer
		// ValidDestinations is a list of end destinations the edge supports. This field is used within determining the path (edge expansion) between two resources.
		ValidDestinations []reflect.Type
	}

	// EdgeKB is a map (knowledge base) of edges and their respective details used to configure ResourceGraphs
	EdgeKB map[Edge]EdgeDetails

	// EdgeExpander is a function used to create the To and From resource and necessary intermediate resources, if any do not exist, to ensure the nodes are in place for correct functionality.
	EdgeExpander func(from, to Resource, dag *ResourceGraph, data EdgeData) error
	// EdgeConfigurer is a function used to configure the To and From resources and necessary dependent resources, to ensure the nodes will guarantee correct functionality.
	EdgeConfigurer func(from, to Resource, dag *ResourceGraph, data EdgeData) error

	// EdgeConstraint is an object defined on EdgeData which can influence the path picked when expansion occurs.
	EdgeConstraint struct {
		// NodeMustExist specifies a list of resources which must exist in the path when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustExist []Resource
		// NodeMustNotExist specifies a list of resources which must not exist when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustNotExist []Resource
	}

	// EdgeData is an object attached to edges in the ResourceGraph to help the knowledge base understand context when performing expansion and configuration tasks
	EdgeData struct {
		// AppName refers to the application name of the global ResourceGraph
		AppName string
		// EnvironmentVaribles specify and environment variables which will need to be configured during the edge expansion process
		EnvironmentVariables []EnvironmentVariable
		// Constraint refers to the EdgeConstraints defined during the edge expansion
		Constraint EdgeConstraint
		// Source refers to the initial source node when edge expansion is called
		Source Resource
		// Destination refers to the initial target node when edge expansion is called
		Destination Resource
	}
)

// GetEdgeDetails takes in a source and target to retrieve the edge details for the given key. Will return nil if no edge exists for the given source and target
func GetEdgeDetails(kb EdgeKB, source reflect.Type, target reflect.Type) EdgeDetails {
	return kb[Edge{From: source, To: target}]
}

// GetEdgesWithSource will return all edges where the source type parameter is the From of the edge
func GetEdgesWithSource(kb EdgeKB, source reflect.Type) []Edge {
	result := []Edge{}
	for edge := range kb {
		if edge.From == source {
			result = append(result, edge)
		}
	}
	return result
}

// GetEdgesWithTarget will return all edges where the target type parameter is the To of the edge
func GetEdgesWithTarget(kb EdgeKB, target reflect.Type) []Edge {
	result := []Edge{}
	for edge := range kb {
		if edge.To == target {
			result = append(result, edge)
		}
	}
	return result
}

// FindPaths takes in a source and destination type and finds all valid paths to get from source to destination.
//
// Find paths does a Depth First Search to search through all edges in the knowledge base.
// The function tracks visited edges to prevent cycles during execution
// It also checks the ValidDestinations for each edge against the original destination node to ensure that the edge is allowed to be used in the instance of the path generation
//
// The method will return all paths found
func FindPaths(kb EdgeKB, source reflect.Type, dest reflect.Type) [][]Edge {
	zap.S().Debugf("Finding Paths from %s -> %s", source.String(), dest.String())
	result := [][]Edge{}
	visitedEdges := map[reflect.Type]bool{}
	stack := []Edge{}
	findPaths(kb, source, dest, stack, visitedEdges, &result)
	return result
}

// findPaths performs the recursive calls of the parent FindPath function
//
// It works under the assumption that an edge is bidirectional and uses the edges ValidDestinations field to determine when that assumption is incorrect
func findPaths(kb EdgeKB, source reflect.Type, dest reflect.Type, stack []Edge, visited map[reflect.Type]bool, result *[][]Edge) {
	visited[source] = true
	if source == dest {
		*result = append(*result, stack)
	} else {
		for _, e := range GetEdgesWithSource(kb, source) {
			if e.From == source && !visited[e.To] && isValidForPath(kb, e, dest) {
				findPaths(kb, e.To, dest, append(stack, e), visited, result)
			}
		}
		for _, e := range GetEdgesWithTarget(kb, source) {
			if e.To == source && !visited[e.From] && isValidForPath(kb, e, dest) {
				findPaths(kb, e.From, dest, append(stack, e), visited, result)
			}
		}
	}
	delete(visited, source)
}

// isValidForPath determines if an edge is valid for an instance of path generation.
//
// The criteria is:
//   - if there is no expansion function and no ValidDestinations, assume all destinations are valid.
//   - otherwise check to see if the path generations destination is valid for the edge
func isValidForPath(kb EdgeKB, edge Edge, dest reflect.Type) bool {
	edgeDetail := kb[edge]
	if edgeDetail.ExpansionFunc == nil || len(edgeDetail.ValidDestinations) == 0 {
		return true
	}
	isDestValid := false
	for _, validDest := range edgeDetail.ValidDestinations {
		if validDest == dest {
			isDestValid = true
		}
	}
	return isDestValid
}

// ExpandEdges performs calculations to determine the proper path to be inserted into the ResourceGraph.
//
// The workflow of the edge expansion is as follows:
//   - Find all valid paths from the dependencies source node to the dependencies target node
//   - Check each of the valid paths against constraints passed in on the edge data
//   - At this point we should only have 1 valid path (If we have more than 1 edge choose direct connection otherwise error)
//   - Iterate through each edge in path calling expansion function on edge
func ExpandEdges(kb EdgeKB, dag *ResourceGraph) (err error) {
	zap.S().Debug("Expanding Edges")
	var merr multierr.Error
	// It does not matter what order we go in as each edge should be expanded independently. They can still reuse resources since the create methods should be idempotent if resources are the same.
	for _, dep := range dag.ListDependencies() {
		zap.S().Debug("Expanding Edge for %s -> %s", dep.Source.Id().String(), dep.Destination.Id().String())

		// We want to retrieve the edge data from the edge in the resource graph to use during expansion
		edgeData := EdgeData{}
		data, ok := dep.Properties.Data.(EdgeData)
		if !ok && dep.Properties.Data != nil {
			merr.Append(fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format", dep.Source.Id().String(), dep.Destination.Id().String()))
		} else if dep.Properties.Data != nil {
			edgeData = data
		}
		// Find all possible paths given the initial source and destination node
		paths := FindPaths(kb, reflect.TypeOf(dep.Source), reflect.TypeOf(dep.Destination))
		validPaths := [][]Edge{}
		for _, path := range paths {

			// Ensure that the path satisfies the NodeMustExist edge constraint
			if edgeData.Constraint.NodeMustExist != nil {
				nodeFound := false
				for _, res := range path {
					for _, mustExistRes := range edgeData.Constraint.NodeMustExist {
						if res.From == reflect.TypeOf(mustExistRes) || res.To == reflect.TypeOf(mustExistRes) {
							nodeFound = true
						}
					}
				}
				if !nodeFound {
					continue
				}
			}

			// Ensure that the path satisfies the NodeMustNotExist edge constraint
			if edgeData.Constraint.NodeMustNotExist != nil {
				nodeFound := false
				for _, res := range path {
					for _, mustNotExistRes := range edgeData.Constraint.NodeMustNotExist {
						if res.From == reflect.TypeOf(mustNotExistRes) || res.To == reflect.TypeOf(mustNotExistRes) {
							nodeFound = true
						}
					}
				}
				if nodeFound {
					continue
				}
			}
			validPaths = append(validPaths, path)
		}

		zap.S().Debugf("Found valid paths %s", validPaths)
		var validPath []Edge

		// If we have more than 1 valid path we will always default to the direct path if it exists, otherwise we will raise an error since we cannot determine which path we are supposed to use.
		if len(validPaths) > 1 {
			for _, p := range validPaths {
				if len(p) == 2 {
					zap.S().Debug("Defaulting to direct path")
					validPath = p
				}
			}
			if len(validPath) == 0 {
				merr.Append(fmt.Errorf("found multiple paths which satisfy constraints for edge %s -> %s. \n Paths: %s", dep.Source.Id().String(), dep.Destination.Id().String(), validPaths))
			}
		} else {
			validPath = validPaths[0]

			// If the valid path is not the original direct path, we want to remove the initial direct dependency so we can fill in the new edges with intermediate nodes
			if len(validPath) > 2 {
				zap.S().Debugf("Removing dependency from %s -> %s", dep.Source.Id().String(), dep.Destination.Id().String())
				err := dag.RemoveDependency(dep.Source.Id().String(), dep.Destination.Id().String())
				if err != nil {
					merr.Append(err)
					continue
				}
			}

			// resourceCache is used to always pass the graphs nodes into the Expand functions if they exist. We do this so that we operate on nodes which already exist
			resourceCache := map[reflect.Type]Resource{}
			for _, edge := range validPath {
				from := edge.From
				to := edge.To
				edgeDetail := GetEdgeDetails(kb, from, to)
				fromNode := reflect.New(from.Elem()).Interface().(Resource)
				if res, ok := resourceCache[from]; ok {
					fromNode = res
				}
				if from == reflect.TypeOf(dep.Source) {
					fromNode = dep.Source
				}
				toNode := reflect.New(to.Elem()).Interface().(Resource)
				if to == reflect.TypeOf(dep.Destination) {
					toNode = dep.Destination
				}
				if res, ok := resourceCache[to]; ok {
					toNode = res
				}

				if edgeDetail.ExpansionFunc != nil {
					err := edgeDetail.ExpansionFunc(fromNode, toNode, dag, edgeData)
					merr.Append(err)
				}

				resourceCache[from] = fromNode
				fromNodeInGraph := dag.GetResource(fromNode.Id())
				if fromNodeInGraph != nil {
					resourceCache[from] = fromNodeInGraph
				}
				resourceCache[to] = toNode
				toNodeInGraph := dag.GetResource(toNode.Id())
				if fromNodeInGraph != nil {
					resourceCache[to] = toNodeInGraph
				}
			}
		}

	}
	return merr.ErrOrNil()
}

// ConfigureFromEdgeData calls each edges configure function.
func ConfigureFromEdgeData(kb EdgeKB, dag *ResourceGraph) (err error) {
	zap.S().Debug("Configuring Edges")
	var merr multierr.Error
	for _, dep := range dag.ListDependencies() {
		zap.S().Debugf("Configuring Edge for %s -> %s", dep.Source.Id().String(), dep.Destination.Id().String())
		to := reflect.TypeOf(dep.Source)
		from := reflect.TypeOf(dep.Destination)
		edgeData := EdgeData{}
		data, ok := dep.Properties.Data.(EdgeData)
		if !ok && dep.Properties.Data != nil {
			merr.Append(fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format", dep.Source.Id().String(), dep.Destination.Id().String()))
		} else if dep.Properties.Data != nil {
			edgeData = data
		}
		edgeDetail := GetEdgeDetails(kb, to, from)
		if edgeDetail.Configure != nil {
			err := edgeDetail.Configure(dep.Source, dep.Destination, dag, edgeData)
			merr.Append(err)
		}
	}
	return merr.ErrOrNil()
}
