package operational_rule

import (
	"sort"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ResourcePlacer interface {
		PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep,
			availableResources []*construct.Resource, numNeeded *int) (Result, error)
		SetCtx(ctx OperationalRuleContext)
	}

	SpreadPlacer struct {
		ctx OperationalRuleContext
	}

	ClusterPlacer struct {
		ctx OperationalRuleContext
	}

	ClosestPlacer struct {
		ctx OperationalRuleContext
	}
)

var placerMap = map[knowledgebase.SelectionOperator]func() ResourcePlacer{
	knowledgebase.SpreadSelectionOperator:  func() ResourcePlacer { return &SpreadPlacer{} },
	knowledgebase.ClusterSelectionOperator: func() ResourcePlacer { return &ClusterPlacer{} },
	knowledgebase.ClosestSelectionOperator: func() ResourcePlacer { return &ClosestPlacer{} },
}

func (p *SpreadPlacer) PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep,
	availableResources []*construct.Resource, numNeeded *int) (Result, error) {
	// if we get the spread operator our logic goes as follows:
	// If there is only one resource available, do not place in that resource and instead create a new one
	// If there are multiple available, find the one with the least connections to the same resource in question and use that

	result := Result{}
	if len(availableResources) <= 1 {
		// If there is only one resource available, do not place in that resource and instead create a new one
		return result, nil
	}
	if *numNeeded == 0 {
		return result, nil
	}
	mapOfConnections, err := p.ctx.findNumConnectionsToTypeForAvailableResources(step, availableResources, resource.ID)
	if err != nil {
		return result, err
	}
	numConnectionsArray := sortNumConnectionsMap(mapOfConnections)

	for _, numConnections := range numConnectionsArray {
		for _, availableResource := range mapOfConnections[numConnections] {
			edge, err := p.ctx.addDependencyForDirection(step, resource, availableResource)
			if err != nil {
				return result, err
			}
			result.AddedDependencies = append(result.AddedDependencies, edge)
			*numNeeded--
			if *numNeeded == 0 {
				return result, nil
			}
		}
	}
	return result, nil
}

func (p *SpreadPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func (p *ClusterPlacer) PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep,
	availableResources []*construct.Resource, numNeeded *int) (Result, error) {
	// if we get the cluster operator our logic goes as follows:
	// Place in the resource which has the most connections to the same resource in question
	result := Result{}
	mapOfConnections, err := p.ctx.findNumConnectionsToTypeForAvailableResources(step, availableResources, resource.ID)
	if err != nil {
		return result, err
	}
	numConnectionsArray := sortNumConnectionsMap(mapOfConnections)
	sort.Sort(sort.Reverse(sort.IntSlice(numConnectionsArray)))
	for _, numConnections := range numConnectionsArray {
		for _, availableResource := range mapOfConnections[numConnections] {
			edge, err := p.ctx.addDependencyForDirection(step, resource, availableResource)
			if err != nil {
				return result, err
			}
			result.AddedDependencies = append(result.AddedDependencies, edge)
			*numNeeded--
			if *numNeeded == 0 {
				return result, nil
			}
		}
	}
	return result, nil
}

func (p *ClusterPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func (p ClosestPlacer) PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep,
	availableResources []*construct.Resource, numNeeded *int) (Result, error) {
	// if we get the closest operator our logic goes as follows:
	// find the closest available resource in terms of functional distance and use that
	result := Result{}
	lengthMap := map[int][]*construct.Resource{}
	for _, availableResource := range availableResources {
		var path []construct.ResourceId
		var err error
		var length int
		undirectedGraph, err := p.buildUndirectedGraph()
		if err != nil {
			return result, err
		}
		if step.Direction == knowledgebase.DirectionDownstream {
			path, err = graph.ShortestPath(undirectedGraph, resource.ID, availableResource.ID)
		} else {
			path, err = graph.ShortestPath(undirectedGraph, availableResource.ID, resource.ID)
		}
		if err != nil {
			return result, err
		}
		for _, resource := range path {
			if p.ctx.Solution.KnowledgeBase().GetFunctionality(resource) != knowledgebase.Unknown {
				length++
			}
		}
		lengthMap[length] = append(lengthMap[length], availableResource)
	}
	sortedLengthList := sortNumConnectionsMap(lengthMap)
	for _, length := range sortedLengthList {
		for _, availableResource := range lengthMap[length] {
			edge, err := p.ctx.addDependencyForDirection(step, resource, availableResource)
			if err != nil {
				return result, err
			}
			result.AddedDependencies = append(result.AddedDependencies, edge)
			*numNeeded--
			if *numNeeded == 0 {
				return result, nil
			}
		}
	}
	return result, nil
}

func (p *ClosestPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func (p *ClosestPlacer) buildUndirectedGraph() (construct.Graph, error) {
	undirected := graph.New(
		func(r *construct.Resource) construct.ResourceId {
			return r.ID
		},
	)
	err := undirected.AddVerticesFrom(p.ctx.Solution.RawView())
	if err != nil {
		return nil, err
	}
	err = undirected.AddEdgesFrom(p.ctx.Solution.RawView())
	if err != nil {
		return nil, err
	}
	return undirected, nil
}

func (ctx OperationalRuleContext) findNumConnectionsToTypeForAvailableResources(
	step knowledgebase.OperationalStep, availableResources []*construct.Resource,
	resource construct.ResourceId) (map[int][]*construct.Resource, error) {

	mapOfConnections := map[int][]*construct.Resource{}
	// If there are multiple available, find the one with the least connections to the same resource in question and use that
	for _, availableResource := range availableResources {
		var err error
		var connections []construct.ResourceId
		if step.Direction == knowledgebase.DirectionDownstream {
			connections, err = solution_context.Upstream(ctx.Solution, availableResource.ID,
				knowledgebase.FirstFunctionalLayer)
		} else {
			connections, err = solution_context.Downstream(ctx.Solution, availableResource.ID,
				knowledgebase.FirstFunctionalLayer)
		}
		var connectionsOfType []construct.ResourceId
		for _, connection := range connections {
			if connection.QualifiedTypeName() == resource.QualifiedTypeName() {
				connectionsOfType = append(connectionsOfType, connection)
			}
		}
		if err != nil {
			return mapOfConnections, err
		}
		mapOfConnections[len(connectionsOfType)] = append(mapOfConnections[len(connectionsOfType)], availableResource)
	}
	return mapOfConnections, nil
}

func sortNumConnectionsMap(mapOfConnections map[int][]*construct.Resource) []int {
	numConnectionsArray := []int{}
	for numConnections := range mapOfConnections {
		numConnectionsArray = append(numConnectionsArray, numConnections)
	}
	sort.Ints(numConnectionsArray)
	return numConnectionsArray
}
