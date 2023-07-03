package knowledgebase

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type (
	// Edge defines an entry in a Knowledge base. An Edge represents a valid linking between two types of resources
	Edge struct {
		// Source represents the source resource in the edge
		Source reflect.Type
		// Destination represents the target resource in the edge
		Destination reflect.Type
	}

	// EdgeDetails defines the set of characteristics and edge in the knowledge base contains. The details are used to ensure graph correctness for ResourceGraphs
	EdgeDetails struct {
		// Configure is a function used to configure the To and From resources and necessary dependent resources, to ensure the nodes will guarantee correct functionality.
		Configure ConfigureEdge
		// DirectEdgeOnly signals that the edge cannot be used within constructing other paths and can only be used as a direct edge
		DirectEdgeOnly bool
		// ReverseDirection is specified when the data flow is in the opposite direction of the edge
		// This is used in scenarios where we want to find paths, only allowing specific edges to be bidirectional
		ReverseDirection bool
		// DeletetionDependent is used to specify edges which should not influence the deletion criteria of a resource
		// a true value specifies the target being deleted is dependent on the source and do not need to depend on satisfication of the deletion criteria to attempt to delete the true source of the edge.
		DeletetionDependent bool
		//Reuse tells us whether we can reuse an upstream or downstream resource during path selection and node creation
		Reuse Reuse
	}

	// Reuse is set to represent an enum of possible reuse cases for edges. The current available options are upstream and downstream
	Reuse string

	// EdgeKB is a map (knowledge base) of edges and their respective details used to configure ResourceGraphs
	EdgeKB map[Edge]EdgeDetails

	// EdgeConfigurer is a function used to configure the To and From resources and necessary dependent resources, to ensure the nodes will guarantee correct functionality.
	ConfigureEdge func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error

	// EdgeConstraint is an object defined on EdgeData which can influence the path picked when expansion occurs.
	EdgeConstraint struct {
		// NodeMustExist specifies a list of resources which must exist in the path when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustExist []core.Resource
		// NodeMustNotExist specifies a list of resources which must not exist when edge expansion occurs. The resources type will be correlated to the types in the generated paths
		NodeMustNotExist []core.Resource
	}

	// EdgeData is an object attached to edges in the ResourceGraph to help the knowledge base understand context when performing expansion and configuration tasks
	EdgeData struct {
		// AppName refers to the application name of the global ResourceGraph
		AppName string
		// EnvironmentVaribles specify and environment variables which will need to be configured during the edge expansion process
		EnvironmentVariables []core.EnvironmentVariable
		// Constraint refers to the EdgeConstraints defined during the edge expansion
		Constraint EdgeConstraint
		// Source refers to the initial source resource node when edge expansion is called
		Source core.Resource
		// Destination refers to the initial target resource node when edge expansion is called
		Destination core.Resource
		// Routes refers to any api routes being satisfied by the edge
		Routes []core.Route
		// SourceRef denotes the source annotation being used during expansion or configuration
		// This is a temporary field due to helm chart being the lowest level of kubernetes resource at the moment
		SourceRef core.BaseConstruct
	}

	Path []Edge
)

const (
	Upstream   Reuse = "upstream"
	Downstream Reuse = "downstream"
)

func NewEdge[Src core.Resource, Dest core.Resource]() Edge {
	var src Src
	var dest Dest
	return Edge{Source: reflect.TypeOf(src), Destination: reflect.TypeOf(dest)}
}

// GetEdge takes in a source and target to retrieve the edge details for the given key. Will return nil if no edge exists for the given source and target
func (kb EdgeKB) GetEdge(source core.Resource, target core.Resource) (EdgeDetails, bool) {
	return kb.GetEdgeDetails(reflect.TypeOf(source), reflect.TypeOf(target))
}

// GetEdgeDetails takes in a source and target to retrieve the edge details for the given key. Will return nil if no edge exists for the given source and target
func (kb EdgeKB) GetEdgeDetails(source reflect.Type, target reflect.Type) (EdgeDetails, bool) {
	detail, found := kb[Edge{Source: source, Destination: target}]
	return detail, found
}

// GetEdgesWithSource will return all edges where the source type parameter is the From of the edge
func (kb EdgeKB) GetEdgesWithSource(source reflect.Type) []Edge {
	result := []Edge{}
	for edge := range kb {
		if edge.Source == source {
			result = append(result, edge)
		}
	}
	return result
}

// GetEdgesWithTarget will return all edges where the target type parameter is the To of the edge
func (kb EdgeKB) GetEdgesWithTarget(target reflect.Type) []Edge {
	result := []Edge{}
	for edge := range kb {
		if edge.Destination == target {
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
func (kb EdgeKB) FindPaths(source core.Resource, dest core.Resource, constraint EdgeConstraint) []Path {
	zap.S().Debugf("Finding Paths from %s -> %s", source.Id(), dest.Id())
	visitedEdges := map[reflect.Type]bool{}
	stack := []Edge{}
	paths := kb.findPaths(reflect.TypeOf(source), reflect.TypeOf(dest), stack, visitedEdges)
	validPaths := []Path{}
	for _, path := range paths {
		// Ensure that the path satisfies the NodeMustExist edge constraint
		if constraint.NodeMustExist != nil {
			nodeFound := false
			for _, res := range path {
				for _, mustExistRes := range constraint.NodeMustExist {
					if res.Source == reflect.TypeOf(mustExistRes) || res.Destination == reflect.TypeOf(mustExistRes) {
						nodeFound = true
					}
				}
			}
			if !nodeFound {
				continue
			}
		}

		// Ensure that the path satisfies the NodeMustNotExist edge constraint
		if constraint.NodeMustNotExist != nil {
			nodeFound := false
			for _, res := range path {
				for _, mustNotExistRes := range constraint.NodeMustNotExist {
					if res.Source == reflect.TypeOf(mustNotExistRes) || res.Destination == reflect.TypeOf(mustNotExistRes) {
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
	return validPaths
}

// findPaths performs the recursive calls of the parent FindPath function
//
// It works under the assumption that an edge is bidirectional and uses the edges ValidDestinations field to determine when that assumption is incorrect
func (kb EdgeKB) findPaths(source reflect.Type, dest reflect.Type, stack []Edge, visited map[reflect.Type]bool) []Path {
	visited[source] = true
	var result []Path

	if source == dest {
		// For resources which can have dependencies between themselves we have to add that path to the stack if it is a valid edge
		if len(stack) == 0 {
			if _, found := kb.GetEdgeDetails(source, dest); found {
				stack = append(stack, Edge{Source: source, Destination: dest})
			}
		}
		if len(stack) != 0 {
			var clonedStack []Edge
			clonedStack = append(clonedStack, stack...)
			result = append(result, clonedStack)
		}
	} else {
		sourceFunctionality := core.GetFunctionality(reflect.New(source.Elem()).Interface().(core.BaseConstruct))
		destFuntionality := core.GetFunctionality(reflect.New(dest.Elem()).Interface().(core.BaseConstruct))
		if len(stack) != 0 && destFuntionality != core.Unknown && sourceFunctionality == destFuntionality {
			fmt.Println(source, dest)
			return result
		}

		// When we are not at the destination we want to recursively call findPaths on all edges which have the source as the current node
		// This is checking all edges which have a direction of From -> To
		for _, e := range kb.GetEdgesWithSource(source) {
			det, _ := kb.GetEdgeDetails(e.Source, e.Destination)
			// Ensure that direct edges cannot contribute to paths. We check if its a direct match for the dest and if not we continue
			if det.DirectEdgeOnly && (len(stack) != 0 || e.Destination != dest) {
				continue
			}
			if !det.ReverseDirection && e.Source == source && !visited[e.Destination] {
				result = append(result, kb.findPaths(e.Destination, dest, append(stack, e), visited)...)
			}
		}
		// When we are not at the destination we want to recursively call findPaths on all edges which have the target as the current node
		// This is checking all edges which have a path direction of To -> From, which is opposite of their dependencies on each other
		//
		// An example of this scenario is in the AWS knowledge base where RdsProxyTarget -> RdsProxy  and RdsProxyTarget -> RdsInstance are valid edges
		// However we would expect the path to be RdsProxy -> RdsProxyTarget -> RdsInstance, so to satisfy understanding the path to connect other nodes, we must understand the direction of both the IaC dependency and data flow dependency
		for _, e := range kb.GetEdgesWithTarget(source) {
			det, _ := kb.GetEdgeDetails(e.Source, e.Destination)
			// Ensure that direct edges cannot contribute to paths. We check if its a direct match for the dest and if not we continue
			if det.DirectEdgeOnly && (len(stack) != 0 || e.Source != dest) {
				continue
			}
			if det.ReverseDirection && e.Destination == source && !visited[e.Source] {
				result = append(result, kb.findPaths(e.Source, dest, append(stack, e), visited)...)
			}
		}
	}
	delete(visited, source)
	return result
}

// FindShortestPath determines the shortest path to get from the dependency's source node to destination node, using the knowledgebase of edges
func (kb EdgeKB) FindShortestPath(dep graph.Edge[core.Resource], constraint EdgeConstraint) (Path, error) {
	validPaths := kb.FindPaths(dep.Source, dep.Destination, constraint)
	var validPath []Edge

	var sameLengthPaths []Path
	// Get the shortest route that satisfied constraints
	for _, path := range validPaths {
		if len(validPath) == 0 {
			validPath = path
		} else if len(path) < len(validPath) {
			validPath = path
			sameLengthPaths = []Path{}
		} else if len(path) == len(validPath) {
			sameLengthPaths = append(sameLengthPaths, path, validPath)
		}
	}
	if len(sameLengthPaths) > 0 {
		return nil, fmt.Errorf("found multiple paths which satisfy constraints for edge %s -> %s and are the same length. \n Paths: %s", dep.Source.Id(), dep.Destination.Id(), sameLengthPaths)
	}

	if len(validPath) == 0 {
		return nil, fmt.Errorf("found no paths which satisfy constraints %s for edge %s -> %s. \n Paths: %s", constraint, dep.Source.Id(), dep.Destination.Id(), validPaths)
	}
	return validPath, nil
}

// ExpandEdges performs calculations to determine the proper path to be inserted into the ResourceGraph.
//
// The workflow of the edge expansion is as follows:
//   - Find shortest path given the constraints on the edge
//   - Iterate through each edge in path creating the resource if necessary
func (kb EdgeKB) ExpandEdge(dep *graph.Edge[core.Resource], dag *core.ResourceGraph) (err error) {

	// It does not matter what order we go in as each edge should be expanded independently. They can still reuse resources since the create methods should be idempotent if resources are the same.
	zap.S().Debugf("Expanding Edge for %s -> %s", dep.Source.Id(), dep.Destination.Id())

	// We want to retrieve the edge data from the edge in the resource graph to use during expansion
	edgeData := EdgeData{}
	data, ok := dep.Properties.Data.(EdgeData)
	if !ok && dep.Properties.Data != nil {
		return fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format during expansion", dep.Source.Id(), dep.Destination.Id())
	} else if dep.Properties.Data != nil {
		edgeData = data
	}
	// We attach the dependencies source and destination nodes for context during expansion
	edgeData.Source = dep.Source
	edgeData.Destination = dep.Destination
	// Find all possible paths given the initial source and destination node
	validPath, err := kb.FindShortestPath(*dep, data.Constraint)
	if err != nil {
		return err
	}
	zap.S().Debugf("Found valid path %s", validPath)
	// resourceCache is used to always pass the graphs nodes into the Expand functions if they exist. We do this so that we operate on nodes which already exist
	resourceCache := map[reflect.Type]core.Resource{}
	var joinedErr error

	name := fmt.Sprintf("%s_%s", dep.Source.Id().Name, dep.Destination.Id().Name)
	for _, edge := range validPath {
		source := edge.Source
		dest := edge.Destination
		edgeDetail, _ := kb.GetEdgeDetails(source, dest)
		sourceNode := resourceCache[source]
		// Determine if the source node is the actual source of the dependency getting expanded
		if source == reflect.TypeOf(dep.Source) {
			sourceNode = dep.Source
		} else if source == reflect.TypeOf(dep.Destination) && edgeDetail.ReverseDirection {
			sourceNode = dep.Destination
		}
		if sourceNode == nil {
			// Create a new interface of the source nodes type if it does not exist
			sourceNode = reflect.New(source.Elem()).Interface().(core.Resource)
			reflect.ValueOf(sourceNode).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s_%s", sourceNode.Id().Type, name)))
			// Determine if the source node is the same type as what is specified in the constraints as must exist
			for _, mustExistRes := range edgeData.Constraint.NodeMustExist {
				if mustExistRes.Id().Type == sourceNode.Id().Type && mustExistRes.Id().Provider == sourceNode.Id().Provider && mustExistRes.Id().Namespace == sourceNode.Id().Namespace {
					sourceNode = mustExistRes
				}
			}
		}

		// Determine if the destination node is the actual destination of the dependency getting expanded
		destNode := resourceCache[dest]
		if dest == reflect.TypeOf(dep.Destination) {
			destNode = dep.Destination
		} else if dest == reflect.TypeOf(dep.Source) && edgeDetail.ReverseDirection {
			destNode = dep.Source
		}

		if destNode == nil {
			// Create a new interface of the destination nodes type if it does not exist
			destNode = reflect.New(dest.Elem()).Interface().(core.Resource)
			reflect.ValueOf(destNode).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s_%s", destNode.Id().Type, name)))
			// Determine if the destination node is the same type as what is specified in the constraints as must exist
			for _, mustExistRes := range edgeData.Constraint.NodeMustExist {
				if mustExistRes.Id().Type == destNode.Id().Type && mustExistRes.Id().Provider == destNode.Id().Provider && mustExistRes.Id().Namespace == destNode.Id().Namespace {
					destNode = mustExistRes
				}
			}
		}

		added := false

		// If the edge specifies that it can reuse upstream or downstream resources, we want to find the first resource which satisfies the reuse criteria and add that as the dependency.
		// If there is no resource that satisfies the reuse criteria, we want to add the original direct dependency
		if edgeDetail.Reuse == Upstream {
			upstreamResources := kb.GetAllTrueDownstream(dep.Source, dag)
			for _, res := range upstreamResources {
				if sourceNode.Id().Type == res.Id().Type {
					dag.AddDependencyWithData(res, destNode, EdgeData{Source: dep.Source, Destination: dep.Destination})
					added = true
				}
			}
		} else if edgeDetail.Reuse == Downstream {
			upstreamResources := kb.GetAllTrueUpstream(dep.Destination, dag)
			for _, res := range upstreamResources {
				if destNode.Id().Type == res.Id().Type {
					dag.AddDependencyWithData(sourceNode, res, EdgeData{Source: dep.Source, Destination: dep.Destination})
					added = true
				}
			}
		}
		if added {
			break
		}

		dag.AddDependencyWithData(sourceNode, destNode, EdgeData{Source: dep.Source, Destination: dep.Destination})

		if sourceNode != nil {
			resourceCache[source] = sourceNode
		}
		sourceNodeInGraph := dag.GetResource(sourceNode.Id())
		if sourceNodeInGraph != nil {
			resourceCache[source] = sourceNodeInGraph
		}
		if destNode != nil {
			resourceCache[dest] = destNode
		}
		destNodeInGraph := dag.GetResource(destNode.Id())
		if destNodeInGraph != nil {
			resourceCache[dest] = destNodeInGraph
		}
	}

	// If the valid path is not the original direct path, we want to remove the initial direct dependency so we can fill in the new edges with intermediate nodes
	if len(validPath) > 1 && joinedErr == nil {
		zap.S().Debugf("Removing dependency from %s -> %s", dep.Source.Id(), dep.Destination.Id())
		err := dag.RemoveDependency(dep.Source.Id(), dep.Destination.Id())
		if err != nil {
			return err
		}

	}
	return joinedErr
}

// ConfigureEdge calls each edge configure function.
func (kb EdgeKB) ConfigureEdge(dep *graph.Edge[core.Resource], dag *core.ResourceGraph) (err error) {
	zap.S().Debugf("Configuring Edge for %s -> %s", dep.Source.Id(), dep.Destination.Id())
	source := reflect.TypeOf(dep.Source)
	destination := reflect.TypeOf(dep.Destination)
	edgeData := EdgeData{}
	data, ok := dep.Properties.Data.(EdgeData)
	if !ok && dep.Properties.Data != nil {
		return fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format during edge configuration", dep.Source.Id(), dep.Destination.Id())
	} else if dep.Properties.Data != nil {
		edgeData = data
	}
	edgeDetail, found := kb.GetEdgeDetails(source, destination)
	if !found {
		return fmt.Errorf("internal error invalid edge for edge %s -> %s (no such edge in Edge KB)", dep.Source.Id(), dep.Destination.Id())
	}
	if edgeDetail.Configure != nil {
		err := edgeDetail.Configure(dep.Source, dep.Destination, dag, edgeData)
		if err != nil {
			return err
		}
	}
	return nil
}

// FindPathsInGraph takes in a source and destination type and finds all valid paths to get from source to destination.
//
// Find paths does a Depth First Search to search through all edges in the knowledge base.
// The function tracks visited edges to prevent cycles during execution
// It also checks the ValidDestinations for each edge against the original destination node to ensure that the edge is allowed to be used in the instance of the path generation
//
// The method will return all paths found
func (kb EdgeKB) FindPathsInGraph(source core.Resource, dest core.Resource, dag *core.ResourceGraph) [][]graph.Edge[core.Resource] {
	zap.S().Debugf("Finding Paths in graph from %s -> %s", source.Id(), dest.Id())
	visitedEdges := map[core.Resource]bool{}
	stack := []graph.Edge[core.Resource]{}
	return kb.findPathsInGraph(source, dest, stack, visitedEdges, dag)
}

// findPathsInGraph performs the recursive calls of the parent FindPath function
//
// It works under the assumption that an edge is bidirectional and uses the edges ValidDestinations field to determine when that assumption is incorrect
func (kb EdgeKB) findPathsInGraph(source, dest core.Resource, stack []graph.Edge[core.Resource], visited map[core.Resource]bool, dag *core.ResourceGraph) (result [][]graph.Edge[core.Resource]) {
	visited[source] = true
	if source == dest {
		if len(stack) != 0 {
			result = append(result, stack)
		}
	} else {
		// When we are not at the destination we want to recursively call findPaths on all edges which have the source as the current node
		// This is checking all edges which have a direction of From -> To
		for _, edge := range dag.GetDownstreamDependencies(source) {
			det, _ := kb.GetEdgeDetails(reflect.TypeOf(edge.Source), reflect.TypeOf(edge.Destination))
			if !det.ReverseDirection && edge.Source == source && !visited[edge.Destination] {
				result = append(result, kb.findPathsInGraph(edge.Destination, dest, append(stack, edge), visited, dag)...)
			}
		}
		// When we are not at the destination we want to recursively call findPaths on all edges which have the target as the current node
		// This is checking all edges which have a path direction of To -> From, which is opposite of their dependencies on each other
		//
		// An example of this scenario is in the AWS knowledge base where RdsProxyTarget -> RdsProxy  and RdsProxyTarget -> RdsInstance are valid edges
		// However we would expect the path to be RdsProxy -> RdsProxyTarget -> RdsInstance, so to satisfy understanding the path to connect other nodes, we must understand the direction of both the IaC dependency and data flow dependency

		for _, edge := range dag.GetUpstreamDependencies(source) {
			det, _ := kb.GetEdgeDetails(reflect.TypeOf(edge.Source), reflect.TypeOf(edge.Destination))
			if det.ReverseDirection && edge.Destination == source && !visited[edge.Source] {
				result = append(result, kb.findPathsInGraph(edge.Source, dest, append(stack, edge), visited, dag)...)
			}
		}
	}
	delete(visited, source)
	return result
}

// GetTrueUpstream takes in a resource and returns all upstream resources which exist in the dag, if their edge does not specify the reverse direction flag in the knowledge base.
// If the edge specifies the reverse direction flag and the resource is downstream, it will be returned as an upstream resource.
func (kb EdgeKB) GetTrueUpstream(source core.Resource, dag *core.ResourceGraph) []core.Resource {
	upstreamResources := []core.Resource{}
	upstreamFromDag := dag.GetUpstreamResources(source)
	for _, res := range upstreamFromDag {
		ed, found := kb.GetEdgeDetails(reflect.TypeOf(res), reflect.TypeOf(source))
		if found && !ed.ReverseDirection {
			upstreamResources = append(upstreamResources, res)
		}
	}
	downstreamFromDag := dag.GetDownstreamResources(source)
	for _, res := range downstreamFromDag {
		ed, found := kb.GetEdgeDetails(reflect.TypeOf(source), reflect.TypeOf(res))
		if found && ed.ReverseDirection {
			upstreamResources = append(upstreamResources, res)
		}
	}
	return upstreamResources
}

// GetTrueUpstream takes in a resource and returns all upstream resources which exist in the dag, if their edge does not specify the reverse direction flag in the knowledge base.
// If the edge specifies the reverse direction flag and the resource is downstream, it will be returned as an upstream resource.
func (kb EdgeKB) GetAllTrueUpstream(source core.Resource, dag *core.ResourceGraph) []core.Resource {
	var upstreams []core.Resource
	upstreamsSet := make(map[core.Resource]struct{})
	for r := range kb.getAllTrueUpstreamResourcesSet(source, dag, upstreamsSet) {
		upstreams = append(upstreams, r)
	}
	return upstreams
}

func (kb EdgeKB) getAllTrueUpstreamResourcesSet(source core.Resource, dag *core.ResourceGraph, upstreams map[core.Resource]struct{}) map[core.Resource]struct{} {
	for _, r := range kb.GetTrueUpstream(source, dag) {
		upstreams[r] = struct{}{}
		kb.getAllTrueUpstreamResourcesSet(r, dag, upstreams)
	}
	return upstreams
}

// GetTrueDownstream takes in a resource and returns all downstream resources which exist in the dag, if their edge does not specify the reverse direction flag in the knowledge base.
// If the edge specifies the reverse direction flag and the resource is upstream, it will be returned as an downstream resource.
func (kb EdgeKB) GetTrueDownstream(source core.Resource, dag *core.ResourceGraph) []core.Resource {
	downstreamResources := []core.Resource{}
	upstreamFromDag := dag.GetUpstreamResources(source)
	for _, res := range upstreamFromDag {
		ed, found := kb.GetEdgeDetails(reflect.TypeOf(res), reflect.TypeOf(source))
		if found && ed.ReverseDirection {
			downstreamResources = append(downstreamResources, res)
		}
	}
	downstreamFromDag := dag.GetDownstreamResources(source)
	for _, res := range downstreamFromDag {
		ed, found := kb.GetEdgeDetails(reflect.TypeOf(source), reflect.TypeOf(res))
		if found && !ed.ReverseDirection {
			downstreamResources = append(downstreamResources, res)
		}
	}
	return downstreamResources
}

// GetAllTrueDownstream takes in a resource and returns all downstream resources which exist in the dag, if their edge does not specify the reverse direction flag in the knowledge base.
// If the edge specifies the reverse direction flag and the resource is upstream, it will be returned as an downstream resource.
func (kb EdgeKB) GetAllTrueDownstream(source core.Resource, dag *core.ResourceGraph) []core.Resource {
	var downstreams []core.Resource
	downstreamsSet := make(map[core.Resource]struct{})
	for r := range kb.getAllTrueDownstreamResourcesSet(source, dag, downstreamsSet) {
		downstreams = append(downstreams, r)
	}
	return downstreams
}

func (kb EdgeKB) getAllTrueDownstreamResourcesSet(source core.Resource, dag *core.ResourceGraph, downstreams map[core.Resource]struct{}) map[core.Resource]struct{} {
	for _, r := range kb.GetTrueUpstream(source, dag) {
		downstreams[r] = struct{}{}
		kb.getAllTrueDownstreamResourcesSet(r, dag, downstreams)
	}
	return downstreams
}
