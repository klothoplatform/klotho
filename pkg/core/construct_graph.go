package core

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"
)

type (
	ConstructGraph struct {
		underlying *graph.Directed[Construct]
	}
)

func NewConstructGraph() *ConstructGraph {
	return &ConstructGraph{
		underlying: graph.NewDirected[Construct](),
	}
}

func (cg *ConstructGraph) GetRoots() []Construct {
	return cg.underlying.Roots()
}

func (cg *ConstructGraph) AddConstruct(construct Construct) {
	zap.S().Infof("Adding resource %s", construct.Id())
	cg.underlying.AddVertex(construct)
}

func (cg *ConstructGraph) AddDependency(source string, dest string) {
	cg.underlying.AddEdge(source, dest)
}

func (cg *ConstructGraph) GetConstruct(key AnnotationKey) Construct {
	return cg.underlying.GetVertex(key.ToId())
}

func (cg *ConstructGraph) ListConstructs() []Construct {
	return cg.underlying.GetAllVertices()
}

func (cg *ConstructGraph) ListDependencies() []graph.Edge[Construct] {
	return cg.underlying.GetAllEdges()
}

func (cg *ConstructGraph) GetDownstreamDependencies(source Construct) []graph.Edge[Construct] {
	return cg.underlying.OutgoingEdges(source)
}

func (cg *ConstructGraph) GetDownstreamConstructs(source Construct) []Construct {
	return cg.underlying.OutgoingVertices(source)
}

func (cg *ConstructGraph) GetUpstreamDependencies(source Construct) []graph.Edge[Construct] {
	return cg.underlying.IncomingEdges(source)
}

func (cg *ConstructGraph) GetUpstreamConstructs(source Construct) []Construct {
	return cg.underlying.IncomingVertices(source)
}

func (cg *ConstructGraph) GetResourcesOfCapability(capability string) (filtered []Construct) {
	vertices := cg.underlying.GetAllVertices()
	for _, v := range vertices {
		fmt.Println(v)
		fmt.Println(capability)
		if v.Provenance().Capability == capability {
			filtered = append(filtered, v)

		}
	}
	return
}

func GetResourcesOfType[T Construct](g *ConstructGraph) (filtered []T) {
	vertices := g.underlying.GetAllVertices()
	for _, v := range vertices {
		if vT, ok := v.(T); ok {
			filtered = append(filtered, vT)
		}
	}
	return
}

func (cg *ConstructGraph) GetExecUnitForPath(fp string) (*ExecutionUnit, File) {
	var best *ExecutionUnit
	var bestFile File
	for _, eu := range GetResourcesOfType[*ExecutionUnit](cg) {
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
