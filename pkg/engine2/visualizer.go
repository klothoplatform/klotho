package engine2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
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
		Resource construct.ResourceId   `yaml:"id"`
		Parent   construct.ResourceId   `yaml:"parent,omitempty"`
		Children []construct.ResourceId `yaml:"children,omitempty"`
	}

	TopologyEdge struct {
		Source construct.ResourceId   `yaml:"source"`
		Target construct.ResourceId   `yaml:"target"`
		Path   []construct.ResourceId `yaml:"path"`
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
		DAG:            ctx.DeploymentGraph(),
	}
	dataflow_topo := &visualizer.File{
		FilenamePrefix: "dataflow-",
		Provider:       "aws",
	}
	var err error
	dataflow_topo.DAG, err = e.GetViewsDag(DataflowView, ctx)
	return []klotho_io.File{iac_topo, dataflow_topo}, err
}

func (e *Engine) GetResourceVizTag(view string, resource construct.ResourceId) Tag {
	template, err := e.Kb.GetResourceTemplate(resource)

	if template == nil || err != nil {
		return NoRenderTag
	}
	tag, found := template.Views[view]
	if !found {
		return NoRenderTag
	}
	return Tag(tag)
}

func (e *Engine) RenderConnection(path []construct.ResourceId) bool {
	for _, res := range path[1 : len(path)-1] {
		tag := e.GetResourceVizTag(string(DataflowView), res)
		if tag == BigIconTag || tag == ParentIconTag {
			return false
		}
	}
	return true
}

func (e *Engine) GetViewsDag(view View, ctx solution_context.SolutionContext) (construct.Graph, error) {
	topo := Topology{Nodes: map[string]*TopologyNode{}}
	viewDag := construct.NewGraph()
	df := ctx.DataflowGraph()

	resources, err := construct.ReverseTopologicalSort(df)
	if err != nil {
		return nil, err
	}
	var errs error
	for _, src := range resources {
		tag := e.GetResourceVizTag(string(DataflowView), src)
		switch tag {
		case ParentIconTag:
			topo.Nodes[src.String()] = &TopologyNode{
				Resource: src,
			}
		case BigIconTag:
			topo.Nodes[src.String()] = &TopologyNode{
				Resource: src,
			}
		case SmallIconTag:
			continue
		case NoRenderTag:
			continue
		default:
			errs = errors.Join(errs, fmt.Errorf("unknown tag %s, for resource %s", tag, src))
			continue
		}
		for _, dst := range resources {
			if src == dst {
				continue
			}
			path, err := graph.ShortestPath(df, src, dst)
			switch {
			case errors.Is(err, graph.ErrTargetNotReachable):
				continue

			case err != nil:
				errs = errors.Join(errs, err)
				continue
			}

			dstTag := e.GetResourceVizTag(string(DataflowView), dst)
			switch dstTag {
			case ParentIconTag:
				if e.RenderConnection(path) {
					topoNode := topo.Nodes[src.String()]
					if !topoNode.Parent.IsZero() {
						currpath, err := graph.ShortestPath(df, src, topoNode.Parent)
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
				if knowledgebase.IsOperationalResourceSideEffect(df, ctx.KnowledgeBase(), src, dst) {
					topoNode := topo.Nodes[src.String()]
					topoNode.Children = append(topoNode.Children, dst)
				}
			case NoRenderTag:
				continue
			default:
				errs = errors.Join(errs, fmt.Errorf("unknown tag %s, for resource %s", dstTag, dst))
			}
		}
	}
	if errs != nil {
		return nil, errs
	}

	for _, node := range topo.Nodes {
		childrenIds := make([]string, len(node.Children))
		for i, child := range node.Children {
			childrenIds[i] = child.String()
		}
		properties := map[string]string{}
		if len(node.Children) > 0 {
			properties["children"] = strings.Join(childrenIds, ",")
		}
		if !node.Parent.IsZero() {
			properties["parent"] = node.Parent.String()
		}
		res, err := df.Vertex(node.Resource)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		errs = errors.Join(errs, viewDag.AddVertex(res, graph.VertexAttributes(properties)))
	}
	if errs != nil {
		return nil, errs
	}

	for _, edge := range topo.Edges {
		pathStrings := make([]string, len(edge.Path))
		for i, res := range edge.Path {
			pathStrings[i] = res.String()
		}
		data := map[string]interface{}{}
		if len(edge.Path) > 0 {
			data["path"] = strings.Join(pathStrings, ",")
		}
		errs = errors.Join(errs, viewDag.AddEdge(edge.Source, edge.Target, graph.EdgeData(data)))
	}
	if errs != nil {
		return nil, errs
	}

	return viewDag, nil
}
