package core

import (
	"os"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
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

func (cg *ConstructGraph) TopologicalSort() ([]string, error) {
	return cg.underlying.VertexIdsInTopologicalOrder()
}

func (cg *ConstructGraph) AddConstruct(construct Construct) {
	zap.S().Infof("Adding resource %s", construct.Id())
	cg.underlying.AddVertex(construct)
}

func (cg *ConstructGraph) AddDependency(source string, dest string) {
	cg.underlying.AddEdge(source, dest)
}

func (cg *ConstructGraph) GetConstruct(key string) Construct {
	return cg.underlying.GetVertex(key)
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

func GetConstruct[T Construct](g *ConstructGraph, key string) (construct T, ok bool) {
	cR := g.GetConstruct(key)
	construct, ok = cR.(T)
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

func CreateConstructGraphFromFile(path string) (*ConstructGraph, error) {

	type ConstructRepresentation struct {
		Kind string `yaml:"kind"`
	}

	type EdgeRepresentation struct {
		Source      string `yaml:"source"`
		Destination string `yaml:"destination"`
	}

	type ConstructGraphRepresentation struct {
		Nodes map[string]ConstructRepresentation `yaml:"nodes"`
		Edges []EdgeRepresentation               `yaml:"edges"`
	}

	graph := NewConstructGraph()
	input := ConstructGraphRepresentation{
		Nodes: map[string]ConstructRepresentation{},
	}
	keys := map[string]AnnotationKey{}

	f, err := os.Open(path)
	if err != nil {
		return graph, err
	}
	defer f.Close() // nolint:errcheck

	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return graph, err
	}

	for id, construct := range input.Nodes {
		switch construct.Kind {
		case "execution_unit":
			unit := &ExecutionUnit{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.ExecutionUnitCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "static_unit":
			unit := &StaticUnit{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.StaticUnitCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "orm":
			unit := &Orm{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "kv":
			unit := &Kv{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "redis_node":
			unit := &RedisNode{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "fs":
			unit := &Fs{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.PersistCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "secret":
			unit := &Config{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.PersistCapability}, Secret: true}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		case "expose":
			unit := &Gateway{AnnotationKey: AnnotationKey{ID: id, Capability: annotation.ExposeCapability}}
			graph.AddConstruct(unit)
			keys[id] = unit.AnnotationKey
		default:
			return graph, errors.Errorf("Unsupported kind %s in construct graph creation", construct.Kind)
		}
	}

	for _, edge := range input.Edges {
		graph.AddDependency(keys[edge.Source].ToId(), keys[edge.Destination].ToId())
	}

	return graph, err
}
