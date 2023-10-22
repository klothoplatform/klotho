package engine2

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	visualizer "github.com/klothoplatform/klotho/pkg/visualizer2"
	"go.uber.org/zap"
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
		Resource construct.ResourceId
		Parent   construct.ResourceId
		Children set.Set[construct.ResourceId]
	}
)
type Topology struct {
	Nodes map[string]*TopologyNode
	Edges map[construct.SimpleEdge]construct.Path
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

func (e *Engine) RenderConnection(path construct.Path) bool {
	for _, res := range path[1 : len(path)-1] {
		tag := e.GetResourceVizTag(string(DataflowView), res)
		if tag == BigIconTag || tag == ParentIconTag {
			return false
		}
	}
	return true
}

func (e *Engine) GetViewsDag(view View, ctx solution_context.SolutionContext) (construct.Graph, error) {
	topo := Topology{
		Nodes: make(map[string]*TopologyNode),
		Edges: make(map[construct.SimpleEdge]construct.Path),
	}
	viewDag := construct.NewGraph()
	df := ctx.DataflowGraph()

	resources, err := construct.ReverseTopologicalSort(df)
	if err != nil {
		return nil, err
	}
	var errs error
	for _, src := range resources {
		node := &TopologyNode{
			Resource: src,
			Children: make(set.Set[construct.ResourceId]),
		}
		tag := e.GetResourceVizTag(string(DataflowView), src)
		switch tag {
		case ParentIconTag, BigIconTag:
			topo.Nodes[src.String()] = node
		case SmallIconTag, NoRenderTag:
			continue
		default:
			errs = errors.Join(errs, fmt.Errorf("unknown tag %s, for resource %s", tag, src))
			continue
		}

		deps, err := construct.DownstreamDependencies(
			df,
			src,
			knowledgebase.DependenciesSkipEdgeLayer(df, e.Kb, src, knowledgebase.FirstFunctionalLayer),
		)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		zap.S().Debugf("%s paths: %d", src, len(deps.Paths))
		sort.Slice(deps.Paths, func(i, j int) bool {
			return len(deps.Paths[i]) < len(deps.Paths[j])
		})
		seenSmall := make(set.Set[construct.ResourceId])
		for _, path := range deps.Paths {
			for i, dst := range path[1:] {
				dstTag := e.GetResourceVizTag(string(DataflowView), dst)
				switch dstTag {
				case ParentIconTag:
					if node.Parent.IsZero() && e.RenderConnection(path[:i+2]) {
						node.Parent = dst
					}
				case BigIconTag:
					if e.RenderConnection(path) {
						edge := construct.SimpleEdge{
							Source: src,
							Target: dst,
						}
						topo.Edges[edge] = path[1 : len(path)-1]
					}
				case SmallIconTag:
					if seenSmall.Contains(dst) {
						continue
					}
					seenSmall.Add(dst)
					if knowledgebase.IsOperationalResourceSideEffect(df, ctx.KnowledgeBase(), src, dst) {
						node.Children.Add(dst)
					}
				case NoRenderTag:
					continue
				default:
					errs = errors.Join(errs, fmt.Errorf("unknown tag %s, for resource %s", dstTag, dst))
				}
			}
		}
	}
	if errs != nil {
		return nil, errs
	}

	for _, node := range topo.Nodes {
		childrenIds := make([]string, len(node.Children))
		children := node.Children.ToSlice()
		sort.Slice(children, func(i, j int) bool {
			return construct.ResourceIdLess(children[i], children[j])
		})
		for i, child := range children {
			childrenIds[i] = child.String()
		}
		properties := map[string]string{}
		if len(node.Children) > 0 {
			properties[string(visualizer.ChildrenKey)] = strings.Join(childrenIds, ",")
		}
		if !node.Parent.IsZero() {
			properties[string(visualizer.ParentKey)] = node.Parent.String()
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

	for edge, path := range topo.Edges {
		pathStrings := make([]string, len(path))
		for i, res := range path {
			pathStrings[i] = res.String()
		}
		data := map[string]interface{}{}
		if len(path) > 0 {
			data["path"] = strings.Join(pathStrings, ",")
		}
		errs = errors.Join(errs, viewDag.AddEdge(edge.Source, edge.Target, graph.EdgeData(data)))
	}
	if errs != nil {
		return nil, errs
	}

	return viewDag, nil
}
