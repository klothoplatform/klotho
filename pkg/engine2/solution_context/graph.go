package solution_context

import (
	"github.com/klothoplatform/klotho/pkg/construct"
)

func (c SolutionContext) ListResources() []construct.Resource {
	return c.dataflowGraph.ListResources()
}

func (c SolutionContext) AddResource(resource construct.Resource) {
	if c.dataflowGraph.GetResource(resource.Id()) == nil {
		c.dataflowGraph.AddResource(resource)
		c.deploymentGraph.AddResource(resource)

		c.RecordDecision(AddResourceDecision{
			Resource: resource.Id(),
		})
	}

}

func (c SolutionContext) AddDependency(from, to construct.Resource) error {
	c.AddResource(from)
	c.AddResource(to)

	c.dataflowGraph.AddDependency(from, to)
	et := c.kb.GetEdgeTemplate(from.Id(), to.Id())
	if et.DeploymentOrderReversed {
		c.deploymentGraph.AddDependency(to, from)
	} else {
		c.deploymentGraph.AddDependency(from, to)
	}
	c.RecordDecision(AddDependencyDecision{
		From: from.Id(),
		To:   to.Id(),
	})
	return c.addPath(from, to)
}

func (c SolutionContext) RemoveResource(resource construct.Resource, explicit bool) error {
	// TODO: Reconciliation should happen here
	// TODO: Find all references of the id in other resources and remove it
	if c.dataflowGraph.GetResource(resource.Id()) == nil {
		err := c.dataflowGraph.RemoveResource(resource)
		if err != nil {
			return err
		}
		err = c.deploymentGraph.RemoveResource(resource)
		if err != nil {
			return err
		}

		c.RecordDecision(RemoveResourceDecision{
			Resource: resource.Id(),
		})
	}
	return nil
}

func (c SolutionContext) RemoveDependency(source construct.ResourceId, destination construct.ResourceId) error {
	if c.dataflowGraph.GetDependency(source, destination) != nil {
		err := c.dataflowGraph.RemoveDependency(source, destination)
		if err != nil {
			return err
		}

		et := c.kb.GetEdgeTemplate(source, destination)
		if et.DeploymentOrderReversed {
			c.deploymentGraph.RemoveDependency(destination, source)
		} else {
			c.deploymentGraph.RemoveDependency(source, destination)
		}
		c.RecordDecision(RemoveDependencyDecision{
			From: source,
			To:   destination,
		})
	}
	return nil
}

func (ctx SolutionContext) GetResource(id construct.ResourceId) construct.Resource {
	return ctx.dataflowGraph.GetResource(id)
}

func (ctx SolutionContext) GetFunctionalDownstreamResourcesOfType(resource construct.Resource, qualifiedType construct.ResourceId) []construct.Resource {
	var result []construct.Resource
	if !ctx.kb.HasFunctionalPath(resource.Id(), qualifiedType) {
		return result
	}
	for _, res := range ctx.GetFunctionalDownstreamResources(resource) {
		if res.Id().QualifiedTypeName() == qualifiedType.QualifiedTypeName() {
			result = append(result, res)
		}
	}
	return result
}

func (ctx SolutionContext) GetFunctionalDownstreamResources(resource construct.Resource) []construct.Resource {
	var result []construct.Resource
	for _, res := range ctx.dataflowGraph.GetAllDownstreamResources(resource) {
		paths, err := ctx.dataflowGraph.AllPaths(resource.Id(), res.Id())
		if err != nil {
			continue
		}
	PATHS:
		for _, path := range paths {
			for _, res := range path {
				if ctx.kb.GetFunctionality(res.Id()) != construct.Unknown {
					continue PATHS
				}
			}
			result = append(result, res)
		}
	}
	return result
}

func (ctx SolutionContext) GetFunctionalUpstreamResourcesOfType(resource construct.Resource, qualifiedType construct.ResourceId) []construct.Resource {
	var result []construct.Resource
	if !ctx.kb.HasFunctionalPath(resource.Id(), qualifiedType) {
		return result
	}
	for _, res := range ctx.GetFunctionalUpstreamResources(resource) {
		if res.Id().QualifiedTypeName() == qualifiedType.QualifiedTypeName() {
			result = append(result, res)
		}
	}
	return result
}

func (ctx SolutionContext) GetFunctionalUpstreamResources(resource construct.Resource) []construct.Resource {
	var result []construct.Resource
	for _, res := range ctx.dataflowGraph.GetAllUpstreamResources(resource) {
		paths, err := ctx.dataflowGraph.AllPaths(resource.Id(), res.Id())
		if err != nil {
			continue
		}
	PATHS:
		for _, path := range paths {
			for _, res := range path {
				if ctx.kb.GetFunctionality(res.Id()) != construct.Unknown {
					continue PATHS
				}
			}
			result = append(result, res)
		}
	}
	return result
}

func (c SolutionContext) ReplaceResourceId(oldId construct.ResourceId, resource construct.Resource) error {
	err := c.dataflowGraph.ReplaceConstructId(oldId, resource)
	if err != nil {
		return err
	}
	err = c.deploymentGraph.ReplaceConstructId(oldId, resource)
	if err != nil {
		return err
	}
	// TODO: Find all references of the old id in other resources and replace it
	return nil
}
