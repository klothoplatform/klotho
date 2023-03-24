package core

import (
	"github.com/klothoplatform/klotho/pkg/graph"
)

type (
	ResourceGraph struct {
		Underlying *graph.Directed[Resource]
	}
)

func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		Underlying: graph.NewDirected[Resource](),
	}
}

func (cg *ResourceGraph) AddResource(resource Resource) {
	cg.Underlying.AddVertex(resource)
}

func (cg *ResourceGraph) AddDependency(source Resource, dest Resource) {
	cg.Underlying.AddEdge(source.Id(), dest.Id())
}

func (cg *ResourceGraph) GetResource(id string) Resource {
	return cg.Underlying.GetVertex(id)
}

func (cg *ResourceGraph) ListConstructs() []Resource {
	return cg.Underlying.GetAllVertices()
}

func (cg *ResourceGraph) ListDependencies() []graph.Edge[Resource] {
	return cg.Underlying.GetAllEdges()
}
