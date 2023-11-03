package engine2

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_eval"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
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
			Parent:   e.getParentFromNamespace(src, resources),
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
			for _, dst := range path[1:] {
				dstTag := e.GetResourceVizTag(string(DataflowView), dst)
				switch dstTag {
				case ParentIconTag:
					hasPath, err := HasPath(topo, ctx, src, dst)
					if err != nil {
						errs = errors.Join(errs, err)
					}
					if node.Parent.IsZero() && hasPath {
						node.Parent = dst
					}
				case BigIconTag:
					hasPath, err := HasPath(topo, ctx, src, dst)
					if err != nil {
						errs = errors.Join(errs, err)
					}
					if hasPath {
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
					isSideEffect, err := knowledgebase.IsOperationalResourceSideEffect(df, ctx.KnowledgeBase(), src, dst)
					if err != nil {
						errs = errors.Join(errs, err)
						continue
					}
					if isSideEffect {
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

	// Remove edges between children and parents
	for _, node := range topo.Nodes {
		if !node.Parent.IsZero() {
			delete(topo.Edges, construct.SimpleEdge{Source: node.Resource, Target: node.Parent})
			delete(topo.Edges, construct.SimpleEdge{Source: node.Parent, Target: node.Resource})
		}
		for edge := range topo.Edges {
			if edge.Source == node.Parent || edge.Target == node.Parent {
				delete(topo.Edges, edge)
				delete(topo.Edges, edge)
			}
		}
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

func (e *Engine) getParentFromNamespace(resource construct.ResourceId, resources []construct.ResourceId) construct.ResourceId {
	if resource.Namespace != "" {
		for _, potentialParent := range resources {
			if potentialParent.Provider == resource.Provider && potentialParent.Name == resource.Namespace && e.GetResourceVizTag(string(DataflowView), potentialParent) == ParentIconTag {
				return potentialParent
			}
		}
	}
	return construct.ResourceId{}
}

func HasPath(topo Topology, sol solution_context.SolutionContext, source, target construct.ResourceId) (bool, error) {
	var errs error
	pathsCache := map[construct.SimpleEdge][][]construct.ResourceId{}
	pathSatisfactions, err := sol.KnowledgeBase().GetPathSatisfactionsFromEdge(source, target)
	if err != nil {
		return false, err
	}
	sourceRes, err := sol.RawView().Vertex(source)
	if err != nil {
		return false, fmt.Errorf("has path could not find source resource %s: %w", source, err)
	}
	targetRes, err := sol.RawView().Vertex(target)
	if err != nil {
		return false, fmt.Errorf("has path could not find target resource %s: %w", target, err)
	}
	edge := construct.ResourceEdge{Source: sourceRes, Target: targetRes}
	for _, satisfaction := range pathSatisfactions {
		expansions, err := operational_eval.DeterminePathSatisfactionInputs(sol, satisfaction, edge)
		if err != nil {
			return false, err
		}
		for _, expansion := range expansions {
			simple := construct.SimpleEdge{Source: expansion.Dep.Source.ID, Target: expansion.Dep.Target.ID}
			paths, found := pathsCache[simple]
			if !found {
				var err error
				paths, err = graph.AllPathsBetween(sol.RawView(), expansion.Dep.Source.ID, expansion.Dep.Target.ID)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
				pathsCache[simple] = paths
			}
			if len(paths) == 0 {
				return false, nil
			}
			containedClassification := false
			if expansion.Classification != "" {
			PATHS:
				for _, path := range paths {
					for i, res := range path {
						if i != 0 && i < len(path)-1 && topo.Nodes[res.String()] != nil {
							continue PATHS
						}
					}
					if path_selection.PathSatisfiesClassification(sol.KnowledgeBase(), path, expansion.Classification) {
						containedClassification = true
						break
					}
				}
			} else {
				containedClassification = true
			}
			if !containedClassification {
				return false, nil
			}
		}
	}
	return true, nil
}
