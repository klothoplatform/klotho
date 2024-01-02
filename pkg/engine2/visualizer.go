package engine2

import (
	"errors"
	"fmt"
	"math"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
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

func (e *Engine) VisualizeViews(ctx solution_context.SolutionContext) ([]klotho_io.File, error) {
	iac_topo := &visualizer.File{
		FilenamePrefix: "iac-",
		Provider:       "aws",
	}
	dataflow_topo := &visualizer.File{
		FilenamePrefix: "dataflow-",
		Provider:       "aws",
	}
	var err error
	iac_topo.Graph, err = visualizer.ConstructToVis(ctx.DeploymentGraph())
	if err != nil {
		return nil, err
	}
	dataflow_topo.Graph, err = e.GetViewsDag(DataflowView, ctx)
	return []klotho_io.File{iac_topo, dataflow_topo}, err
}

func GetResourceVizTag(kb knowledgebase.TemplateKB, view View, resource construct.ResourceId) Tag {
	template, err := kb.GetResourceTemplate(resource)

	if template == nil || err != nil {
		return NoRenderTag
	}
	tag, found := template.Views[string(view)]
	if !found {
		return NoRenderTag
	}
	return Tag(tag)
}

func (e *Engine) GetViewsDag(view View, sol solution_context.SolutionContext) (visualizer.VisGraph, error) {
	viewDag := visualizer.NewVisGraph()
	var graph construct.Graph
	if view == IACView {
		graph = sol.DeploymentGraph()
	} else {
		graph = sol.DataflowGraph()
	}

	undirected := construct.NewGraphWithOptions()
	err := undirected.AddVerticesFrom(graph)
	if err != nil {
		return nil, fmt.Errorf("could not copy vertices for undirected: %w", err)
	}
	err = undirected.AddEdgesFrom(graph)
	if err != nil {
		return nil, fmt.Errorf("could not copy edges for undirected: %w", err)
	}

	ids, err := construct.ReverseTopologicalSort(graph)
	if err != nil {
		return nil, err
	}

	var errs error

	// First pass gets all the vertices (groups or big icons)
	for _, id := range ids {
		var err error
		switch tag := GetResourceVizTag(e.Kb, view, id); tag {
		case NoRenderTag:
			continue
		case ParentIconTag, BigIconTag:
			err = viewDag.AddVertex(&visualizer.VisResource{
				ID:       id,
				Children: make(set.Set[construct.ResourceId]),
				Tag:      string(tag),
			})
		}
		errs = errors.Join(errs, err)
	}
	if errs != nil {
		return nil, errs
	}

	// Second pass sets up the small icons & edges between big icons
	for _, id := range ids {
		switch tag := GetResourceVizTag(e.Kb, view, id); tag {
		case NoRenderTag:
			continue
		case ParentIconTag:
			continue
		case BigIconTag:
			err := e.handleBigIcon(sol, view, undirected, viewDag, id)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle big icon %s: %w", id, err))
			}
		case SmallIconTag:
			err := e.handleSmallIcon(view, undirected, viewDag, id)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle small icon %s: %w", id, err))
			}
		default:
			errs = errors.Join(errs, fmt.Errorf("unknown tag %s", tag))
		}
	}

	return viewDag, errs
}

// handleSmallIcon finds big icons to attach this resource to. It always adds to big icons for which it is
// in the glue layer. It also adds to the big icon that is closest to the resource (if there is one).
func (e *Engine) handleSmallIcon(
	view View,
	g construct.Graph,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) error {
	ids, err := construct.TopologicalSort(viewDag)
	if err != nil {
		return err
	}
	pather, err := construct.ShortestPaths(g, id, construct.DontSkipEdges)
	if err != nil {
		return err
	}
	glueIds, err := knowledgebase.Downstream(g, e.Kb, id, knowledgebase.ResourceGlueLayer)
	if err != nil {
		return err
	}
	glue := set.SetOf(glueIds...)
	var errs error
	var bestParent *visualizer.VisResource
	bestParentWeight := math.MaxInt32
	for _, candidate := range ids {
		// If the resource is in the glue layer, add it
		if glue.Contains(candidate) {
			parent, err := viewDag.Vertex(candidate)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			parent.Children.Add(id)
		}

		// Even if it was in the glue layer, continue calculating the best parent so we don't accidentally
		// attribute it to a worse parent.
		path, err := pather.ShortestPath(candidate)
		if errors.Is(err, graph.ErrTargetNotReachable) {
			continue
		} else if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		weight, err := graph_addons.PathWeight(g, graph_addons.Path[construct.ResourceId](path))
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if weight < bestParentWeight {
			bestParentWeight = weight
			bestParent, err = viewDag.Vertex(candidate)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		}
	}
	if errs != nil {
		return errs
	}
	if bestParent != nil {
		bestParent.Children.Add(id)
	}
	return nil
}

// handleBigIcon sets the parent of the big icon if there is a group it should be added to and
// adds edges to any other big icons based on having the proper connections (network & permissions).
func (e *Engine) handleBigIcon(
	sol solution_context.SolutionContext,
	view View,
	undirected construct.Graph,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) error {
	source, err := viewDag.Vertex(id)
	if err != nil {
		return err
	}
	parent, err := e.findParent(view, undirected, viewDag, id)
	if err != nil {
		return err
	}
	source.Parent = parent

	targets, err := construct.TopologicalSort(viewDag)
	if err != nil {
		return err
	}

	var errs error
	for _, target := range targets {
		if target == id {
			continue
		}
		if tag := GetResourceVizTag(e.Kb, view, target); tag != BigIconTag {
			continue
		}

		hasPath, err := hasVisPath(sol, view, id, target)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if hasPath {
			errs = errors.Join(errs, viewDag.AddEdge(id, target))
		}
	}
	return err
}

func (e *Engine) findParent(
	view View,
	undirected construct.Graph,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) (bestParent construct.ResourceId, err error) {
	ids, err := construct.TopologicalSort(viewDag)
	if err != nil {
		return
	}
	pather, err := construct.ShortestPaths(undirected, id, construct.DontSkipEdges)
	if err != nil {
		return
	}
	bestParentWeight := math.MaxInt32
	var errs error
candidateLoop:
	for _, id := range ids {
		if GetResourceVizTag(e.Kb, view, id) != ParentIconTag {
			continue
		}
		path, err := pather.ShortestPath(id)
		if errors.Is(err, graph.ErrTargetNotReachable) {
			continue
		} else if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, pathElem := range path[1 : len(path)-1] {
			pathTmpl, err := e.Kb.GetResourceTemplate(pathElem)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			// Don't cross functional boundaries for parent attribution
			if pathTmpl.GetFunctionality() != knowledgebase.Unknown {
				continue candidateLoop
			}
		}
		weight, err := graph_addons.PathWeight(undirected, graph_addons.Path[construct.ResourceId](path))
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if weight < bestParentWeight {
			bestParentWeight = weight
			bestParent = id
		}
	}
	err = errs
	return
}

func hasVisPath(sol solution_context.SolutionContext, view View, source, target construct.ResourceId) (bool, error) {
	srcTemplate, err := sol.KnowledgeBase().GetResourceTemplate(source)
	if err != nil || srcTemplate == nil {
		return false, fmt.Errorf("has path could not find source resource %s: %w", source, err)
	}
	targetTemplate, err := sol.KnowledgeBase().GetResourceTemplate(target)
	if err != nil || targetTemplate == nil {
		return false, fmt.Errorf("has path could not find target resource %s: %w", target, err)
	}
	if len(targetTemplate.PathSatisfaction.AsTarget) == 0 || len(srcTemplate.PathSatisfaction.AsSource) == 0 {
		return false, nil
	}
	sourceRes, err := sol.RawView().Vertex(source)
	if err != nil {
		return false, fmt.Errorf("has path could not find source resource %s: %w", source, err)
	}
	targetRes, err := sol.RawView().Vertex(target)
	if err != nil {
		return false, fmt.Errorf("has path could not find target resource %s: %w", target, err)
	}

	consumed, err := knowledgebase.HasConsumedFromResource(
		sourceRes,
		targetRes,
		solution_context.DynamicCtx(sol),
	)
	if err != nil {
		return false, err
	}
	if !consumed {
		return false, nil
	}
	return checkPaths(sol, view, source, target)
}

func checkPaths(sol solution_context.SolutionContext, view View, source, target construct.ResourceId) (bool, error) {
	paths, err := path_selection.GetPaths(
		sol,
		source,
		target,
		func(source, target construct.ResourceId, path []construct.ResourceId) bool {
			for _, res := range path[1 : len(path)-1] {
				switch GetResourceVizTag(sol.KnowledgeBase(), view, res) {
				case BigIconTag, ParentIconTag:
					// Don't consider paths that go through big/parent icons
					return false
				}
			}
			return true
		},
		true,
	)
	return len(paths) > 0, err
}
