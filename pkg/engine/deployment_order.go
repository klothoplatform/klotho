package engine

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

func (e *Engine) GetDeploymentOrderGraph(dataflow *core.ResourceGraph) *core.ResourceGraph {
	deploymentOrderGraph := core.NewResourceGraph()
	for _, resource := range dataflow.ListResources() {
		deploymentOrderGraph.AddResource(resource)
	}
	for _, dep := range dataflow.ListDependencies() {
		edge, _ := e.KnowledgeBase.GetEdge(dep.Source, dep.Destination)
		if edge.DeploymentOrderReversed {
			deploymentOrderGraph.AddDependency(dep.Destination, dep.Source)
		} else {
			deploymentOrderGraph.AddDependency(dep.Source, dep.Destination)
		}
	}
	return deploymentOrderGraph
}
