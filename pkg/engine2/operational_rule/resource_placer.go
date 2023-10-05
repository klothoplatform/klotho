package operational_rule

import (
	"sort"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	ResourcePlacer interface {
		PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep, availableResources []*construct.Resource, numNeeded *int) error
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
	availableResources []*construct.Resource, numNeeded *int) error {
	// if we get the spread operator our logic goes as follows:
	// If there is only one resource available, do not place in that resource and instead create a new one
	// If there are multiple available, find the one with the least connections to the same resource in question and use that

	if len(availableResources) <= 1 {
		// If there is only one resource available, do not place in that resource and instead create a new one
		return nil
	}

	mapOfConnections, err := p.ctx.findNumConnectionsToTypeForAvailableResources(step, availableResources, resource)
	if err != nil {
		return err
	}
	numConnectionsArray := sortNumConnectionsMap(mapOfConnections)

	for _, numConnections := range numConnectionsArray {
		for _, availableResource := range mapOfConnections[numConnections] {
			err := p.ctx.Graph.AddDependency(resource, availableResource)
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

func (p *ClusterPlacer) PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep,
	availableResources []*construct.Resource, numNeeded *int) error {
	// if we get the cluster operator our logic goes as follows:
	// Place in the resource which has the most connections to the same resource in question
	mapOfConnections, err := p.ctx.findNumConnectionsToTypeForAvailableResources(step, availableResources, resource)
	if err != nil {
		return err
	}
	numConnectionsArray := sortNumConnectionsMap(mapOfConnections)
	sort.Sort(sort.Reverse(sort.IntSlice(numConnectionsArray)))
	for _, numConnections := range numConnectionsArray {
		for _, availableResource := range mapOfConnections[numConnections] {
			err := p.ctx.addDependencyForDirection(&step, resource, availableResource)
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

func (p ClosestPlacer) PlaceResources(resource *construct.Resource, step knowledgebase.OperationalStep,
	availableResources []*construct.Resource, numNeeded *int) error {
	// if we get the closest operator our logic goes as follows:
	// find the closest available resource in terms of functional distance and use that
	lengthMap := map[int][]*construct.Resource{}
	for _, availableResource := range availableResources {
		var path []*construct.Resource
		var err error
		var length int
		if step.Direction == knowledgebase.Downstream {
			path, err = p.ctx.Graph.ShortestPath(resource.ID, availableResource.ID)
		} else {
			path, err = p.ctx.Graph.ShortestPath(availableResource.ID, resource.ID)
		}
		if err != nil {
			return err
		}
		for _, resource := range path {
			if p.ctx.KB.GetFunctionality(resource.ID) != knowledgebase.Unknown {
				length++
			}
		}
		lengthMap[length] = append(lengthMap[length], availableResource)
	}
	sortedLengthList := sortNumConnectionsMap(lengthMap)
	for _, length := range sortedLengthList {
		for _, availableResource := range lengthMap[length] {
			err := p.ctx.addDependencyForDirection(&step, resource, availableResource)
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

func (p *ClosestPlacer) SetCtx(ctx OperationalRuleContext) {
	p.ctx = ctx
}

func (ctx OperationalRuleContext) findNumConnectionsToTypeForAvailableResources(
	step knowledgebase.OperationalStep, availableResources []*construct.Resource,
	resource *construct.Resource) (map[int][]*construct.Resource, error) {

	mapOfConnections := map[int][]*construct.Resource{}
	// If there are multiple available, find the one with the least connections to the same resource in question and use that
	for _, availableResource := range availableResources {
		var err error
		var connections []*construct.Resource
		if step.Direction == knowledgebase.Downstream {
			connections, err = ctx.Graph.DownstreamOfType(availableResource, 3, resource.ID.QualifiedTypeName())
		} else {
			connections, err = ctx.Graph.UpstreamOfType(availableResource, 3, resource.ID.QualifiedTypeName())
		}
		if err != nil {
			return mapOfConnections, err
		}
		mapOfConnections[len(connections)] = append(mapOfConnections[len(connections)], availableResource)
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
