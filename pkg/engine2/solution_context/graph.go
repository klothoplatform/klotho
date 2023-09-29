package solution_context

import (
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	// Layer defines how to search for relationships between resources when looking for what is upstream and downstream of a resource
	Layer int
)

const (
	// Layer1 represents any unique resources the target resource needs to be operational
	Layer1 Layer = 1
	// Layer2 represents all upstream/downstream resources that represent glue. This will not include any other functional resources and will stop searching paths once a functional resource is reached.
	Layer2 Layer = 2
	// Layer3 represents all upstream/downstream resources that represent glue and the first functional resource in other paths from the target resource
	Layer3 Layer = 3
	// Layer4 represents all upstream/downstream resources that exist in the graph
	Layer4 Layer = 4
)

func (c SolutionContext) ListResources() ([]*construct.Resource, error) {
	var resources []*construct.Resource
	resourceIds, err := construct.ToplogicalSort(c.dataflowGraph)
	if err != nil {
		return nil, err
	}
	for _, res := range resourceIds {
		res, err := c.dataflowGraph.Vertex(res)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	return resources, nil
}

func (c SolutionContext) AddResource(resource *construct.Resource) error {
	return c.addResource(resource, true)
}

func (c SolutionContext) addResource(resource *construct.Resource, makeOperational bool) error {
	res, err := c.GetResource(resource.ID)
	if err != graph.ErrVertexNotFound && err != nil {
		return fmt.Errorf("error getting resource %s, when trying to add resource", resource.ID)
	}
	if res == nil {
		err := c.dataflowGraph.AddVertex(resource)
		if err != nil {
			return fmt.Errorf("error adding resource %s", resource.ID)
		}
		err = c.deploymentGraph.AddVertex(resource)
		if err != nil {
			return fmt.Errorf("error adding resource %s", resource.ID)
		}
		c.RecordDecision(AddResourceDecision{
			Resource: resource.ID,
		})
		if makeOperational {
			return c.nodeMakeOperational(resource)
		}
	}
	return nil
}

func (c SolutionContext) AddDependency(from, to *construct.Resource) error {
	return c.addDependency(from, to, true)
}

func (c SolutionContext) addDependency(from, to *construct.Resource, makeOperational bool) error {
	err := c.addResource(from, makeOperational)
	if err != nil {
		return fmt.Errorf("error while adding dependency from %s to %s: %v", from.ID, to.ID, err)
	}
	err = c.addResource(to, makeOperational)
	if err != nil {
		return fmt.Errorf("error while adding dependency from %s to %s: %v", from.ID, to.ID, err)
	}
	err = c.dataflowGraph.AddEdge(from.ID, to.ID)
	if err != nil {
		return fmt.Errorf("error while adding dependency from %s to %s: %v", from.ID, to.ID, err)
	}
	et := c.kb.GetEdgeTemplate(from.ID, to.ID)
	if et == nil {
		return fmt.Errorf("edge template not found for %s to %s", from.ID, to.ID)
	}
	if et.DeploymentOrderReversed {
		err = c.deploymentGraph.AddEdge(to.ID, from.ID)
		if err != nil {
			return err
		}
	} else {
		err = c.deploymentGraph.AddEdge(from.ID, to.ID)
		if err != nil {
			return err
		}
	}
	c.RecordDecision(AddDependencyDecision{
		From: from.ID,
		To:   to.ID,
	})
	if !makeOperational {
		return nil
	}
	return c.addPath(from, to)
}

func (c SolutionContext) GetResource(resource construct.ResourceId) (*construct.Resource, error) {
	return c.dataflowGraph.Vertex(resource)
}

func (c SolutionContext) GetDependency(from, to construct.ResourceId) (graph.Edge[*construct.Resource], error) {
	return c.dataflowGraph.Edge(from, to)
}

func (c SolutionContext) RemoveResource(resource *construct.Resource, explicit bool) error {
	// TODO: Find all references of the id in other resources and remove it

	res, err := c.GetResource(resource.ID)
	if err != nil {
		return err
	}
	if res == nil {
		upstreamNodes, err := c.DirectUpstreamResources(resource.ID)
		if err != nil {
			return err
		}
		downstreamNodes, err := c.DirectDownstreamResources(resource.ID)
		if err != nil {
			return err
		}
		if !c.canDeleteResource(resource, explicit, upstreamNodes, downstreamNodes) {
			return nil
		}

		if c.kb.GetFunctionality(resource.ID) == knowledgebase.Unknown {
			err := c.reconnectFunctionalResources(resource)
			if err != nil {
				return err
			}
		}

		err = c.dataflowGraph.RemoveVertex(resource.ID)
		if err != nil {
			return err
		}
		err = c.deploymentGraph.RemoveVertex(resource.ID)
		if err != nil {
			return err
		}

		c.RecordDecision(RemoveResourceDecision{
			Resource: resource.ID,
		})

		for _, res := range upstreamNodes {
			err = c.RemoveResource(res, false)
			if err != nil {
				return err
			}
		}
		for _, res := range downstreamNodes {
			err = c.RemoveResource(res, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c SolutionContext) RemoveDependency(source construct.ResourceId, destination construct.ResourceId) error {
	_, err := c.GetDependency(source, destination)
	if err == graph.ErrEdgeNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	err = c.dataflowGraph.RemoveEdge(source, destination)
	if err != nil {
		return err
	}

	et := c.kb.GetEdgeTemplate(source, destination)
	if et.DeploymentOrderReversed {
		err = c.deploymentGraph.RemoveEdge(destination, source)
		if err != nil {
			return err
		}
	} else {
		err = c.deploymentGraph.RemoveEdge(source, destination)
		if err != nil {
			return err
		}
	}
	c.RecordDecision(RemoveDependencyDecision{
		From: source,
		To:   destination,
	})

	return nil
}

func (ctx SolutionContext) Downstream(resource *construct.Resource, layer int) ([]*construct.Resource, error) {
	ids, err := construct.AllDownstreamDependencies(ctx.dataflowGraph, resource.ID)
	if err != nil {
		return nil, err
	}
	var resources []*construct.Resource
	for _, id := range ids {
		res, err := ctx.dataflowGraph.Vertex(id)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	var result []*construct.Resource
	switch layer {
	case int(Layer1):
		for _, res := range resources {
			_, err := ctx.dataflowGraph.Edge(resource.ID, res.ID)
			if err != nil {
				continue
			}
			if ctx.IsOperationalResourceSideEffect(resource, res) {
				result = append(result, res)
			}
			return result, nil
		}
	case int(Layer2):
		for _, res := range resources {
			if ctx.kb.GetFunctionality(res.ID) == knowledgebase.Unknown && ctx.isDownstreamWithinFunctionalBoundary(resource, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case int(Layer3):
		for _, res := range resources {
			if ctx.isDownstreamWithinFunctionalBoundary(resource, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case int(Layer4):
		return resources, nil
	}
	return nil, nil
}

func (ctx SolutionContext) DownstreamOfType(resource *construct.Resource, layer int, qualifiedType string) ([]*construct.Resource, error) {
	resources, err := ctx.Downstream(resource, layer)
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, res := range resources {
		if res.ID.QualifiedTypeName() == qualifiedType {
			result = append(result, res)
		}
	}
	return result, nil
}

func (ctx SolutionContext) DownstreamFunctional(resource *construct.Resource) ([]*construct.Resource, error) {
	resources, err := ctx.Downstream(resource, int(Layer3))
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, res := range resources {
		if ctx.kb.GetFunctionality(res.ID) != knowledgebase.Unknown {
			result = append(result, res)
		}
	}
	return result, nil
}

func (ctx SolutionContext) isDownstreamWithinFunctionalBoundary(resource *construct.Resource, downstream *construct.Resource) bool {
	paths, err := ctx.AllPaths(resource.ID, downstream.ID)
	if err != nil {
		return false
	}
	for _, path := range paths {
		for _, res := range path {
			if ctx.kb.GetFunctionality(res.ID) != knowledgebase.Unknown {
				return false
			}
		}
	}
	return true
}

func (ctx SolutionContext) Upstream(resource *construct.Resource, layer int) ([]*construct.Resource, error) {
	ids, err := construct.AllUpstreamDependencies(ctx.dataflowGraph, resource.ID)
	if err != nil {
		return nil, err
	}
	var resources []*construct.Resource
	for _, id := range ids {
		res, err := ctx.dataflowGraph.Vertex(id)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	var result []*construct.Resource
	switch layer {
	case int(Layer1):
		for _, res := range resources {
			_, err := ctx.dataflowGraph.Edge(res.ID, resource.ID)
			if err != nil {
				continue
			}
			if ctx.IsOperationalResourceSideEffect(resource, res) {
				result = append(result, res)
			}
			return result, nil
		}
	case int(Layer2):
		for _, res := range resources {
			if ctx.kb.GetFunctionality(res.ID) == knowledgebase.Unknown && ctx.isUpstreamWithinFunctionalBoundary(resource, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case int(Layer3):
		for _, res := range resources {
			if ctx.isUpstreamWithinFunctionalBoundary(resource, res) {
				result = append(result, res)
			}
		}
		return result, nil
	case int(Layer4):
		return resources, nil
	}
	return nil, nil
}

func (ctx SolutionContext) UpstreamOfType(resource *construct.Resource, layer int, qualifiedType string) ([]*construct.Resource, error) {
	resources, err := ctx.Upstream(resource, layer)
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, res := range resources {
		if res.ID.QualifiedTypeName() == qualifiedType {
			result = append(result, res)
		}
	}
	return result, nil
}

func (ctx SolutionContext) UpstreamFunctional(resource *construct.Resource) ([]*construct.Resource, error) {
	resources, err := ctx.Upstream(resource, int(Layer3))
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, res := range resources {
		if ctx.kb.GetFunctionality(res.ID) != knowledgebase.Unknown {
			result = append(result, res)
		}
	}
	return result, nil
}

func (ctx SolutionContext) isUpstreamWithinFunctionalBoundary(resource *construct.Resource, downstream *construct.Resource) bool {
	paths, err := ctx.AllPaths(downstream.ID, resource.ID)
	if err != nil {
		return false
	}
	for _, path := range paths {
		for _, res := range path {
			if ctx.kb.GetFunctionality(res.ID) != knowledgebase.Unknown {
				return false
			}
		}
	}
	return true
}

// GetDirectUpstreamResources returns all resources which are directly upstream of the given resource
func (ctx SolutionContext) DirectUpstreamResources(id construct.ResourceId) ([]*construct.Resource, error) {
	ids, err := construct.DirectUpstreamDependencies(ctx.dataflowGraph, id)
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, id := range ids {
		res, err := ctx.dataflowGraph.Vertex(id)
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

// GetDirectDownstreamResources returns all resources which are directly downstream of the given resource
func (ctx SolutionContext) DirectDownstreamResources(id construct.ResourceId) ([]*construct.Resource, error) {
	ids, err := construct.DirectDownstreamDependencies(ctx.dataflowGraph, id)
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, id := range ids {
		res, err := ctx.dataflowGraph.Vertex(id)
		if err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}

func (c SolutionContext) ReplaceResourceId(oldId construct.ResourceId, resource *construct.Resource) error {
	err := c.AddResource(resource)
	if err != nil {
		return err
	}

	directUpstream, err := c.DirectUpstreamResources(oldId)
	if err != nil {
		return err
	}
	for _, res := range directUpstream {
		err = c.AddDependency(res, resource)
		if err != nil {
			return err
		}
	}

	directDownstream, err := c.DirectDownstreamResources(oldId)
	if err != nil {
		return err
	}
	for _, res := range directDownstream {
		err = c.AddDependency(resource, res)
		if err != nil {
			return err
		}
	}
	// TODO: Find all references of the old id in other resources and replace it
	return c.RemoveResource(&construct.Resource{ID: oldId}, true)
}

func (c SolutionContext) AllPaths(source construct.ResourceId, destination construct.ResourceId) ([][]*construct.Resource, error) {
	paths, err := graph.AllPathsBetween(c.dataflowGraph, source, destination)
	if err != nil {
		return nil, err
	}
	var result [][]*construct.Resource
	for _, path := range paths {
		var pathResult []*construct.Resource
		for _, res := range path {
			resource, err := c.GetResource(res)
			if err != nil {
				return nil, err
			}
			pathResult = append(pathResult, resource)
		}
		result = append(result, pathResult)
	}
	return result, nil
}

func (c SolutionContext) ShortestPath(source construct.ResourceId, destination construct.ResourceId) ([]*construct.Resource, error) {
	path, err := graph.ShortestPath(c.dataflowGraph, source, destination)
	if err != nil {
		return nil, err
	}
	var result []*construct.Resource
	for _, res := range path {
		resource, err := c.GetResource(res)
		if err != nil {
			return nil, err
		}
		result = append(result, resource)
	}
	return result, nil
}
