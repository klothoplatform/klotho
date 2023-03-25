package core

import (
	"github.com/klothoplatform/klotho/pkg/graph"
)

type (
	ResourceGraph struct {
		underlying *graph.Directed[Resource]
	}
)

func NewResourceGraph() *ResourceGraph {
	return &ResourceGraph{
		underlying: graph.NewDirected[Resource](),
	}
}

func (cg *ResourceGraph) AddResource(resource Resource) {
	cg.underlying.AddVertex(resource)
}

func (cg *ResourceGraph) AddDependency(source Resource, dest Resource) {
	cg.underlying.AddEdge(source.Id(), dest.Id())
}

func (cg *ResourceGraph) GetResource(id string) Resource {
	return cg.underlying.GetVertex(id)
}

func (cg *ResourceGraph) GetDependency(source string, target string) graph.Edge[Resource] {
	return cg.underlying.GetEdge(source, target)
}

func (cg *ResourceGraph) ListConstructs() []Resource {
	return cg.underlying.GetAllVertices()
}

func (cg *ResourceGraph) ListDependencies() []graph.Edge[Resource] {
	return cg.underlying.GetAllEdges()
}
