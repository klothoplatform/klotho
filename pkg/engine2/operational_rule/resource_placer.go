package operational_rule

import (
	"errors"
	"math"
	"sort"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ResourcePlacer interface {
		PlaceResources(
			resource *construct.Resource,
			step knowledgebase.OperationalStep,
			availableResources []*construct.Resource,
			numNeeded *int,
		) error

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

func (p *SpreadPlacer) PlaceResources(
	resource *construct.Resource,
	step knowledgebase.OperationalStep,
	availableResources []*construct.Resource,
	numNeeded *int,
) error {
	// if we get the spread operator our logic goes as follows:
	// If there is only one resource available, do not place in that resource and instead create a new one
	// If there are multiple available, find the one with the least connections to the same resource in question and use that

	if len(availableResources) <= 1 {
		// If there is only one resource available, do not place in that resource and instead create a new one
		return nil
	}
	if *numNeeded == 0 {
		return nil
	}
	mapOfConnections, err := p.ctx.findNumConnectionsToTypeForAvailableResources(step, availableResources, resource.ID)
	if err != nil {
		return err
	}
	numConnectionsArray := sortNumConnectionsMap(mapOfConnections)

	for _, numConnections := range numConnectionsArray {
		for _, availableResource := range mapOfConnections[numConnections] {
			err := p.ctx.addDependencyForDirection(step, resource, availableResource)
			if err != nil {
				return err
			}
			*numNeeded--
			if *numNeeded == 0 {
				return nil
			}
		}
	}
	return nil
}

func (p *SpreadPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func (p *ClusterPlacer) PlaceResources(
	resource *construct.Resource,
	step knowledgebase.OperationalStep,
	availableResources []*construct.Resource,
	numNeeded *int,
) error {
	// if we get the cluster operator our logic goes as follows:
	// Place in the resource which has the most connections to the same resource in question
	mapOfConnections, err := p.ctx.findNumConnectionsToTypeForAvailableResources(step, availableResources, resource.ID)
	if err != nil {
		return err
	}
	numConnectionsArray := sortNumConnectionsMap(mapOfConnections)
	sort.Sort(sort.Reverse(sort.IntSlice(numConnectionsArray)))
	for _, numConnections := range numConnectionsArray {
		for _, availableResource := range mapOfConnections[numConnections] {
			err := p.ctx.addDependencyForDirection(step, resource, availableResource)
			if err != nil {
				return err
			}
			*numNeeded--
			if *numNeeded == 0 {
				return nil
			}
		}
	}
	return nil
}

func (p *ClusterPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func (p ClosestPlacer) PlaceResources(
	resource *construct.Resource,
	step knowledgebase.OperationalStep,
	availableResources []*construct.Resource,
	numNeeded *int,
) error {
	// if we get the closest operator our logic goes as follows:
	// find the closest available resource in terms of functional distance and use that
	if *numNeeded == 0 {
		return nil
	}
	resourceDepths := make(map[construct.ResourceId]int)
	undirectedGraph, err := BuildUndirectedGraph(p.ctx.Solution)
	if err != nil {
		return err
	}
	pather, err := construct.ShortestPaths(undirectedGraph, resource.ID, construct.DontSkipEdges)
	if err != nil {
		return err
	}
	for _, availableResource := range availableResources {
		path, err := pather.ShortestPath(availableResource.ID)
		if err != nil && !errors.Is(err, graph.ErrTargetNotReachable) {
			return err
		}
		// If the target isnt reachable then we want to make it so that it is the longest possible path
		if path == nil {
			resourceDepths[availableResource.ID] = math.MaxInt64
			continue
		}
		length := 0
		for i := range path {
			if i == 0 {
				continue
			}
			edge, _ := undirectedGraph.Edge(path[i-1], path[i])
			length += edge.Properties.Weight
		}
		resourceDepths[availableResource.ID] = length
	}
	sort.SliceStable(availableResources, func(i, j int) bool {
		return resourceDepths[availableResources[i].ID] < resourceDepths[availableResources[j].ID]
	})
	num := *numNeeded
	if num > len(availableResources) || num < 0 {
		num = len(availableResources)
	}
	for _, availableResource := range availableResources[:num] {
		err := p.ctx.addDependencyForDirection(step, resource, availableResource)
		if err != nil {
			return err
		}
	}
	*numNeeded -= num
	return nil
}

func (p *ClosestPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func BuildUndirectedGraph(ctx solution_context.SolutionContext) (construct.Graph, error) {
	undirected := graph.NewWithStore(
		construct.ResourceHasher,
		graph_addons.NewMemoryStore[construct.ResourceId, *construct.Resource](),
	)
	err := undirected.AddVerticesFrom(ctx.RawView())
	if err != nil {
		return nil, err
	}
	edges, err := ctx.RawView().Edges()
	if err != nil {
		return nil, err
	}
	for _, e := range edges {
		weight := 1
		// increase weights for edges that are connected to a functional resource
		if knowledgebase.GetFunctionality(ctx.KnowledgeBase(), e.Source) != knowledgebase.Unknown {
			weight = 1000
		} else if knowledgebase.GetFunctionality(ctx.KnowledgeBase(), e.Target) != knowledgebase.Unknown {
			weight = 1000
		}
		err := undirected.AddEdge(e.Source, e.Target, graph.EdgeWeight(weight))
		if err != nil {
			return nil, err
		}
	}
	return undirected, nil
}

func (ctx OperationalRuleContext) findNumConnectionsToTypeForAvailableResources(
	step knowledgebase.OperationalStep,
	availableResources []*construct.Resource,
	resource construct.ResourceId,
) (map[int][]*construct.Resource, error) {

	mapOfConnections := map[int][]*construct.Resource{}
	// If there are multiple available, find the one with the least connections to the same resource in question and use that
	for _, availableResource := range availableResources {
		var err error
		var connections []construct.ResourceId
		// We will look to see what direct dependencies are already existing in the same direction as the rule
		// if we dont only look at direct, we risk getting incorrect results if the resource can have non functional connections
		if step.Direction == knowledgebase.DirectionDownstream {
			connections, err = solution_context.Upstream(ctx.Solution, availableResource.ID,
				knowledgebase.ResourceDirectLayer)
		} else {
			connections, err = solution_context.Downstream(ctx.Solution, availableResource.ID,
				knowledgebase.ResourceDirectLayer)
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
