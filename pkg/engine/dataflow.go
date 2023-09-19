package engine

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
)

func (e *Engine) RenderConnection(path []construct.Resource) bool {
	for _, res := range path[1 : len(path)-1] {
		tag := e.GetResourceVizTag(string(DataflowView), res)
		if tag == BigIconTag || tag == ParentIconTag {
			return false
		}
	}
	return true
}

type (
	TopologyNode struct {
		Resource construct.Resource   `yaml:"id"`
		Parent   construct.Resource   `yaml:"parent,omitempty"`
		Children []construct.Resource `yaml:"children,omitempty"`
	}

	TopologyEdge struct {
		Source construct.Resource   `yaml:"source"`
		Target construct.Resource   `yaml:"target"`
		Path   []construct.Resource `yaml:"path"`
	}
)
type Topology struct {
	Nodes map[string]*TopologyNode `yaml:"nodes"`
	Edges []TopologyEdge           `yaml:"edges"`
}

func (e *Engine) GetDataFlowDag() *construct.ResourceGraph {
	dag := e.Context.Solution.ResourceGraph
	topo := Topology{Nodes: map[string]*TopologyNode{}}
	dfDag := construct.NewResourceGraph()

	for _, src := range dag.ListResources() {
		tag := e.GetResourceVizTag(string(DataflowView), src)
		switch tag {
		case ParentIconTag:
			topo.Nodes[src.Id().String()] = &TopologyNode{
				Resource: src,
			}
		case BigIconTag:
			topo.Nodes[src.Id().String()] = &TopologyNode{
				Resource: src,
			}
		case SmallIconTag:
			continue
		case NoRenderTag:
			continue
		default:
			panic(fmt.Sprintf("Unknown tag %s, for resource %s", tag, src.Id()))
		}
		for _, dst := range dag.ListResources() {
			if src == dst {
				continue
			}
			dstTag := e.GetResourceVizTag(string(DataflowView), dst)
			path, err := e.Context.Solution.ResourceGraph.ShortestPath(src.Id(), dst.Id())
			if err != nil {
				panic("Error getting shortest path")
			}
			if len(path) == 0 {
				continue
			}
			switch dstTag {
			case ParentIconTag:
				if e.RenderConnection(path) {
					topoNode := topo.Nodes[src.Id().String()]
					if topoNode.Parent != nil {
						currpath, err := e.Context.Solution.ResourceGraph.ShortestPath(src.Id(), topoNode.Parent.Id())
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
				if e.isSideEffect(dag, src, dst) {
					topoNode := topo.Nodes[src.Id().String()]
					topoNode.Children = append(topoNode.Children, dst)
				}
			case NoRenderTag:
				continue
			default:
				panic(fmt.Sprintf("Unknown tag %s, for resource %s", tag, dst.Id()))
			}
		}
	}

	for _, node := range topo.Nodes {
		childrenIds := make([]string, len(node.Children))
		for i, child := range node.Children {
			childrenIds[i] = child.Id().String()
		}
		properties := map[string]string{}
		if len(node.Children) > 0 {
			properties["children"] = strings.Join(childrenIds, ",")
		}
		if node.Parent != nil {
			properties["parent"] = node.Parent.Id().String()
		}
		dfDag.AddResourceWithProperties(node.Resource, properties)
	}

	for _, edge := range topo.Edges {
		pathStrings := make([]string, len(edge.Path))
		for i, res := range edge.Path {
			pathStrings[i] = res.Id().String()
		}
		data := map[string]interface{}{}
		if len(edge.Path) > 0 {
			data["path"] = strings.Join(pathStrings, ",")
		}
		dfDag.AddDependencyWithData(edge.Source, edge.Target, data)
	}

	return dfDag
}
