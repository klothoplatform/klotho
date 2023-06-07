package core

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
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
	OutputGraph struct {
		Resources []ResourceId `yaml:"resources"`
		Edges     []OutputEdge `yaml:"edges"`
	}
)

func NewConstructGraph() *ConstructGraph {
	return &ConstructGraph{
		underlying: graph.NewDirected(func(v BaseConstruct) string {
			return v.Id().String()
		}),
	}
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

func (cg *ConstructGraph) AddDependency(source ResourceId, dest ResourceId) {
	cg.underlying.AddEdge(source.String(), dest.String(), nil)
}

func (cg *ConstructGraph) GetConstruct(key ResourceId) BaseConstruct {
	return cg.underlying.GetVertex(key.String())
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

func (cg *ConstructGraph) GetResourcesOfCapability(capability string) (filtered []Construct) {
	vertices := cg.underlying.GetAllVertices()
	for _, v := range vertices {
		if vCons, ok := v.(Construct); ok && vCons.Provenance().Capability == capability {
			filtered = append(filtered, vCons)
		}
	}
	return
}

func GetConstructsOfType[T Construct](g *ConstructGraph) (filtered []T) {
	vertices := g.underlying.GetAllVertices()
	for _, v := range vertices {
		if vT, ok := v.(T); ok {
			filtered = append(filtered, vT)
		}
	}
	return
}

func GetConstruct[T Construct](g *ConstructGraph, key ResourceId) (construct T, ok bool) {
	cR := g.GetConstruct(key)
	construct, ok = cR.(T)
	return
}

func (cg *ConstructGraph) GetExecUnitForPath(fp string) (*ExecutionUnit, File) {
	var best *ExecutionUnit
	var bestFile File
	for _, eu := range GetConstructsOfType[*ExecutionUnit](cg) {
		f := eu.Get(fp)
		if f != nil {
			astF, ok := f.(*SourceFile)
			if ok && (best == nil || FileExecUnitName(astF) == eu.Provenance().ID) {
				best = eu
				bestFile = f
			}
		}
	}
	return best, bestFile
}

func (cg *ConstructGraph) FindUpstreamGateways(unit *ExecutionUnit) []*Gateway {
	gateways := []*Gateway{}
	vertices := cg.underlying.IncomingVertices(unit)
	for _, v := range vertices {
		gw, ok := v.(*Gateway)
		if ok {
			gateways = append(gateways, gw)
		}
	}
	return gateways
}

func LoadConstructsIntoGraph(input OutputGraph, graph *ConstructGraph) error {

	for _, res := range input.Resources {
		switch res.Type {
		case "execution_unit":
			unit := &ExecutionUnit{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.ExecutionUnitCapability}}
			graph.AddConstruct(unit)
		case "static_unit":
			unit := &StaticUnit{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.StaticUnitCapability}}
			graph.AddConstruct(unit)
		case "orm":
			unit := &Orm{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
		case "kv":
			unit := &Kv{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
		case "redis_node":
			unit := &RedisNode{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
		case "fs":
			unit := &Fs{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
		case "secret":
			unit := &Config{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.PersistCapability}, Secret: true}
			graph.AddConstruct(unit)
		case "expose":
			unit := &Gateway{AnnotationKey: AnnotationKey{ID: res.Name, Capability: annotation.ExposeCapability}}
			graph.AddConstruct(unit)

		}
	}

	for _, edge := range input.Edges {
		graph.AddDependency(edge.Source, edge.Destination)
	}

	return nil
}
