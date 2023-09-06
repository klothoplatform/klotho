package construct

import (
	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type (
	ConstructGraph struct {
		underlying *graph.Directed[BaseConstruct]
	}

	OutputEdge struct {
		Source      ResourceId `yaml:"source"`
		Destination ResourceId `yaml:"destination"`
	}

	ResourceMetadata struct {
		Id       ResourceId    `yaml:"id"`
		Metadata BaseConstruct `yaml:"metadata"`
	}

	InputMetadata struct {
		Id       ResourceId                  `yaml:"id"`
		Metadata map[interface{}]interface{} `yaml:"metadata"`
	}

	OutputGraph struct {
		Resources        []ResourceId       `yaml:"resources"`
		ResourceMetadata []ResourceMetadata `yaml:"resourceMetadata"`
		Edges            []OutputEdge       `yaml:"edges"`
	}

	InputGraph struct {
		Resources        []ResourceId    `yaml:"resources"`
		ResourceMetadata []InputMetadata `yaml:"resourceMetadata"`
		Edges            []OutputEdge    `yaml:"edges"`
	}
)

func (edge OutputEdge) String() string {
	return edge.Source.String() + " -> " + edge.Destination.String()
}

func NewConstructGraph() *ConstructGraph {
	return &ConstructGraph{
		underlying: graph.NewDirected(func(v BaseConstruct) string {
			return v.Id().String()
		}),
	}
}

func (cg *ConstructGraph) Clone() *ConstructGraph {
	newGraph := &ConstructGraph{
		underlying: graph.NewLike(cg.underlying),
	}
	for _, v := range cg.ListConstructs() {
		newGraph.AddConstruct(v)
	}
	for _, dep := range cg.ListDependencies() {
		newGraph.AddDependency(dep.Source.Id(), dep.Destination.Id())
	}
	return newGraph
}

func (cg *ConstructGraph) GetRoots() []BaseConstruct {
	return cg.underlying.Roots()
}

func (cg *ConstructGraph) TopologicalSort() ([]string, error) {
	return cg.underlying.VertexIdsInTopologicalOrder()
}

func (cg *ConstructGraph) AddConstruct(construct BaseConstruct) {
	zap.S().Infof("Adding resource %s", construct.Id())
	cg.underlying.AddVertex(construct)
}

func (cg *ConstructGraph) RemoveConstructAndEdges(construct BaseConstruct) error {
	// Since its a construct we just assume every single edge can be removed
	for _, edge := range cg.GetDownstreamDependencies(construct) {
		err := cg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	for _, edge := range cg.GetUpstreamDependencies(construct) {
		err := cg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	return cg.RemoveConstruct(construct)
}

func (cg *ConstructGraph) ReplaceConstruct(construct BaseConstruct, new BaseConstruct) error {
	cg.AddConstruct(new)
	// Since its a construct we just assume every single edge can be removed
	for _, edge := range cg.GetDownstreamDependencies(construct) {
		cg.AddDependency(new.Id(), edge.Destination.Id())
		err := cg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	for _, edge := range cg.GetUpstreamDependencies(construct) {
		cg.AddDependency(edge.Source.Id(), new.Id())
		err := cg.RemoveDependency(edge.Source.Id(), edge.Destination.Id())
		if err != nil {
			return err
		}
	}
	return cg.RemoveConstruct(construct)
}

func (cg *ConstructGraph) RemoveConstruct(construct BaseConstruct) error {
	zap.S().Infof("Removing construct %s", construct.Id())
	return cg.underlying.RemoveVertex(construct.Id().String())
}

func (cg *ConstructGraph) AddDependency(source ResourceId, dest ResourceId) {
	cg.underlying.AddEdge(source.String(), dest.String(), nil)
}

func (cg *ConstructGraph) AddDependencyWithData(source ResourceId, dest ResourceId, data any) {
	cg.underlying.AddEdge(source.String(), dest.String(), data)
}

func (cg *ConstructGraph) RemoveDependency(source ResourceId, dest ResourceId) error {
	return cg.underlying.RemoveEdge(source.String(), dest.String())
}

func (cg *ConstructGraph) GetConstruct(key ResourceId) BaseConstruct {
	return cg.underlying.GetVertex(key.String())
}
func (cg *ConstructGraph) GetDependency(source ResourceId, target ResourceId) *graph.Edge[BaseConstruct] {
	return cg.underlying.GetEdge(source.String(), target.String())
}

func (cg *ConstructGraph) GetResource(id ResourceId) Resource {
	c := cg.GetConstruct(id)
	if r, ok := c.(Resource); ok {
		return r
	}
	return nil
}

func ListConstructs[C BaseConstruct](cg *ConstructGraph) []C {
	var result []C
	for _, v := range cg.underlying.GetAllVertices() {
		if vc, ok := v.(C); ok {
			result = append(result, vc)
		}
	}
	return result
}

func (cg *ConstructGraph) ListDependencies() []graph.Edge[BaseConstruct] {
	return cg.underlying.GetAllEdges()
}

func (cg *ConstructGraph) ListConstructs() []BaseConstruct {
	return cg.underlying.GetAllVertices()
}

func (cg *ConstructGraph) GetDownstreamDependencies(source BaseConstruct) []graph.Edge[BaseConstruct] {
	return cg.underlying.OutgoingEdges(source)
}

func (cg *ConstructGraph) GetDownstreamConstructs(source BaseConstruct) []BaseConstruct {
	return cg.underlying.OutgoingVertices(source)
}

func (cg *ConstructGraph) GetUpstreamDependencies(source BaseConstruct) []graph.Edge[BaseConstruct] {
	return cg.underlying.IncomingEdges(source)
}

func (cg *ConstructGraph) GetUpstreamConstructs(source BaseConstruct) []BaseConstruct {
	return cg.underlying.IncomingVertices(source)
}

func (cg *ConstructGraph) ShortestPath(source ResourceId, dest ResourceId) ([]BaseConstruct, error) {
	ids, err := cg.underlying.ShortestPath(source.String(), dest.String())
	if err != nil {
		return nil, err
	}
	resources := make([]BaseConstruct, len(ids))
	for i, id := range ids {
		resources[i] = cg.underlying.GetVertex(id)
	}
	return resources, nil
}

func (cg *ConstructGraph) AllPaths(source ResourceId, dest ResourceId) ([][]BaseConstruct, error) {
	paths, err := cg.underlying.AllPaths(source.String(), dest.String())
	if err != nil {
		return nil, err
	}
	var resources [][]BaseConstruct
	for _, path := range paths {
		var p []BaseConstruct
		for _, id := range path {
			p = append(p, cg.underlying.GetVertex(id))
		}
		resources = append(resources, p)
	}
	return resources, nil
}

func (cg *ConstructGraph) GetResourcesOfCapability(capability string) (filtered []Construct) {
	vertices := cg.underlying.GetAllVertices()
	for _, v := range vertices {
		if vCons, ok := v.(Construct); ok && vCons.AnnotationCapability() == capability {
			filtered = append(filtered, vCons)
		}
	}
	return
}

func GetConstructsOfType[T BaseConstruct](g *ConstructGraph) (filtered []T) {
	vertices := g.underlying.GetAllVertices()
	for _, v := range vertices {
		if vT, ok := v.(T); ok {
			filtered = append(filtered, vT)
		}
	}
	return
}

func GetConstruct[T BaseConstruct](g *ConstructGraph, key ResourceId) (construct T, ok bool) {
	cR := g.GetConstruct(key)
	construct, ok = cR.(T)
	return
}
