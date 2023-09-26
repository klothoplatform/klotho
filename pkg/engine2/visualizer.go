package engine2

import (
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	visualizer "github.com/klothoplatform/klotho/pkg/visualizer2"
)

type (
	View string
	Tag  string
)

const (
	DataflowView View = "dataflow"
	IACView      View = "iac"

	ParentIconTag Tag = "parent"
	BigIconTag    Tag = "big"
	SmallIconTag  Tag = "small"
	NoRenderTag   Tag = "no-render"
)

type (
	TopologyNode struct {
		Resource *construct.Resource   `yaml:"id"`
		Parent   *construct.Resource   `yaml:"parent,omitempty"`
		Children []*construct.Resource `yaml:"children,omitempty"`
	}

	TopologyEdge struct {
		Source *construct.Resource   `yaml:"source"`
		Target *construct.Resource   `yaml:"target"`
		Path   []*construct.Resource `yaml:"path"`
	}
)
type Topology struct {
	Nodes map[string]*TopologyNode `yaml:"nodes"`
	Edges []TopologyEdge           `yaml:"edges"`
}

func (e *Engine) VisualizeViews(ctx solution_context.SolutionContext) ([]klotho_io.File, error) {
	iac_topo := &visualizer.File{
		FilenamePrefix: "iac-",
		Provider:       "aws",
		DAG:            ctx.GetDeploymentGraph(),
	}
	dataflow_topo := &visualizer.File{
		FilenamePrefix: "dataflow-",
		Provider:       "aws",
		DAG:            e.GetViewsDag(DataflowView, ctx),
	}
	return []klotho_io.File{iac_topo, dataflow_topo}, nil
}

func (e *Engine) GetResourceVizTag(view string, resource *construct.Resource) Tag {
	template, err := e.Kb.GetResourceTemplate(resource.ID)

	if template == nil || err != nil {
		return NoRenderTag
	}
	tag, found := template.Views[view]
	if !found {
		return NoRenderTag
	}
	return Tag(tag)
}

func (e *Engine) RenderConnection(path []*construct.Resource) bool {
	for _, res := range path[1 : len(path)-1] {
		tag := e.GetResourceVizTag(string(DataflowView), res)
		if tag == BigIconTag || tag == ParentIconTag {
			return false
		}
	}
	return true
}

func (e *Engine) GetViewsDag(view View, ctx solution_context.SolutionContext) construct.Graph {
	topo := Topology{Nodes: map[string]*TopologyNode{}}
	dfDag := construct.NewGraph()

	resources, err := ctx.ListResources()
	if err != nil {
		panic("Error getting resources")
	}
	for _, src := range resources {
		tag := e.GetResourceVizTag(string(DataflowView), src)
		switch tag {
		case ParentIconTag:
			topo.Nodes[src.ID.String()] = &TopologyNode{
				Resource: src,
			}
		case BigIconTag:
			topo.Nodes[src.ID.String()] = &TopologyNode{
				Resource: src,
			}
		case SmallIconTag:
			continue
		case NoRenderTag:
			continue
		default:
			panic(fmt.Sprintf("Unknown tag %s, for resource %s", tag, src.ID))
		}
		for _, dst := range resources {
			if src == dst {
				continue
			}
			dstTag := e.GetResourceVizTag(string(DataflowView), dst)
			path, err := ctx.ShortestPath(src.ID, dst.ID)
			if err != nil {
				panic("Error getting shortest path")
			}
			if len(path) == 0 {
				continue
			}
			switch dstTag {
			case ParentIconTag:
				if e.RenderConnection(path) {
					topoNode := topo.Nodes[src.ID.String()]
					if topoNode.Parent != nil {
						currpath, err := ctx.ShortestPath(src.ID, topoNode.Parent.ID)
						if err != nil {
							panic("Error getting shortest path")
						}
						if len(path) > len(currpath) {
							continue
						}
					}
					topoNode.Parent = dst
				}
			case BigIconTag:
				if e.RenderConnection(path) {
					topo.Edges = append(topo.Edges, TopologyEdge{
						Source: src,
						Target: dst,
						Path:   path[1 : len(path)-1],
					})
				}
			case SmallIconTag:
				if ctx.IsOperationalResourceSideEffect(src, dst) {
					topoNode := topo.Nodes[src.ID.String()]
					topoNode.Children = append(topoNode.Children, dst)
				}
			case NoRenderTag:
				continue
			default:
				panic(fmt.Sprintf("Unknown tag %s, for resource %s", tag, dst.ID))
			}
		}
	}

	for _, node := range topo.Nodes {
		childrenIds := make([]string, len(node.Children))
		for i, child := range node.Children {
			childrenIds[i] = child.ID.String()
		}
		properties := map[string]string{}
		if len(node.Children) > 0 {
			properties["children"] = strings.Join(childrenIds, ",")
		}
		if node.Parent != nil {
			properties["parent"] = node.Parent.ID.String()
		}
		dfDag.AddVertex(node.Resource, graph.VertexAttributes(properties))
	}

	for _, edge := range topo.Edges {
		pathStrings := make([]string, len(edge.Path))
		for i, res := range edge.Path {
			pathStrings[i] = res.ID.String()
		}
		data := map[string]interface{}{}
		if len(edge.Path) > 0 {
			data["path"] = strings.Join(pathStrings, ",")
		}
		dfDag.AddEdge(edge.Source.ID, edge.Target.ID, graph.EdgeData(data))
	}

	return dfDag
}
