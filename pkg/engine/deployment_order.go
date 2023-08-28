package engine

import (
	"github.com/klothoplatform/klotho/pkg/construct"
)

func (e *Engine) GetDeploymentOrderGraph(dataflow *construct.ResourceGraph) *construct.ResourceGraph {
	deploymentOrderGraph := construct.NewResourceGraph()
	for _, resource := range dataflow.ListResources() {
		deploymentOrderGraph.AddResource(resource)
	}
	for _, dep := range dataflow.ListDependencies() {
		edge, _ := e.KnowledgeBase.GetResourceEdge(dep.Source, dep.Destination)
		if edge.DeploymentOrderReversed {
			deploymentOrderGraph.AddDependency(dep.Destination, dep.Source)
		} else {
			deploymentOrderGraph.AddDependency(dep.Source, dep.Destination)
		}
	}
	return deploymentOrderGraph
}
