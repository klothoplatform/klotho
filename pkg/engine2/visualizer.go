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
	var resGraph construct.Graph
	if view == IACView {
		resGraph = sol.DeploymentGraph()
	} else {
		resGraph = sol.DataflowGraph()
	}

	undirected := construct.NewGraphWithOptions()
	err := undirected.AddVerticesFrom(resGraph)
	if err != nil {
		return nil, fmt.Errorf("could not copy vertices for undirected: %w", err)
	}
	err = undirected.AddEdgesFrom(resGraph)
	if err != nil {
		return nil, fmt.Errorf("could not copy edges for undirected: %w", err)
	}

	ids, err := construct.ReverseTopologicalSort(resGraph)
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
			err := e.handleParentIcon(sol, view, viewDag, id)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle parent icon %s: %w", id, err))
			}
		case BigIconTag:
			err := e.handleBigIcon(sol, view, undirected, viewDag, id)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle big icon %s: %w", id, err))
			}
		case SmallIconTag:
			// Small icons don't need special handling, handleBigIcon will look for any relevant small icons to include
		default:
			errs = errors.Join(errs, fmt.Errorf("unknown tag %s", tag))
		}
	}

	return viewDag, errs
}

func (e *Engine) handleParentIcon(
	sol solution_context.SolutionContext,
	view View,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) error {
	this, err := viewDag.Vertex(id)
	if err != nil {
		return err
	}

	parent, err := e.findParent(view, sol, viewDag, id)
	if err != nil {
		return err
	}
	this.Parent = parent

	if err := e.setChildren(sol, view, this); err != nil {
		return err
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
	this, err := viewDag.Vertex(id)
	if err != nil {
		return err
	}

	parent, err := e.findParent(view, sol, viewDag, id)
	if err != nil {
		return err
	}
	this.Parent = parent

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

		paths, err := visPaths(sol, view, id, target)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if len(paths) > 0 {
			allPathResources := make(set.Set[construct.ResourceId])
			for _, path := range paths {
				for _, pathRes := range path[1 : len(path)-1] {
					allPathResources.Add(pathRes)
				}
			}
			errs = errors.Join(errs, viewDag.AddEdge(id, target, graph.EdgeData(visualizer.VisEdgeData{
				PathResources: allPathResources,
			})))
		}
	}
	if errs != nil {
		return errs
	}

	if err := e.setChildren(sol, view, this); err != nil {
		return err
	}

	return nil
}

func (e *Engine) setChildren(sol solution_context.SolutionContext, view View, v *visualizer.VisResource) error {
	glue, err := knowledgebase.Downstream(
		sol.DataflowGraph(),
		sol.KnowledgeBase(),
		v.ID,
		knowledgebase.ResourceLocalLayer,
	)
	if err != nil {
		return fmt.Errorf("failed to get glue layer for %s: %w", v.ID, err)
	}
	for _, glueElem := range glue {
		if GetResourceVizTag(e.Kb, view, glueElem) == SmallIconTag {
			v.Children.Add(glueElem)
		}
	}

	// After glue, also include any resources whose namespace is this resource
	ids, err := construct.TopologicalSort(sol.DataflowGraph())
	if err != nil {
		return err
	}

	var errs error
	for _, id := range ids {
		if id.Namespace == "" {
			continue
		}
		tmpl, err := sol.KnowledgeBase().GetResourceTemplate(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, p := range tmpl.Properties {
			if p.Details().Namespace {
				pres, err := sol.RawView().Vertex(id)
				if err != nil {
					errs = errors.Join(errs, err)
					break
				}
				val, err := pres.GetProperty(p.Details().Path)
				if err != nil {
					errs = errors.Join(errs, err)
					break
				}
				if val == v.ID {
					v.Children.Add(id)
					break
				}
			}
		}
	}

	return nil
}

func (e *Engine) findParent(
	view View,
	sol solution_context.SolutionContext,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) (bestParent construct.ResourceId, err error) {
	if id.Namespace != "" {
		// namespaced resources' parents is always their namespace resource
		tmpl, err := sol.KnowledgeBase().GetResourceTemplate(id)
		if err != nil {
			return bestParent, err
		}
		thisRes, err := sol.RawView().Vertex(id)
		if err != nil {
			return bestParent, err
		}
		for _, p := range tmpl.Properties {
			if !p.Details().Namespace {
				continue
			}
			v, err := thisRes.GetProperty(p.Details().Path)
			if err != nil {
				return bestParent, fmt.Errorf("failed to get namespace property %s: %w", p.Details().Path, err)
			}
			if propId, ok := v.(construct.ResourceId); ok {
				if GetResourceVizTag(e.Kb, view, propId) == ParentIconTag {
					return propId, nil
				}
				// the property isn't shown as a parent (eg. Subnet or ALB Listener), so roll it up to the next parent
				return e.findParent(view, sol, viewDag, propId)
			} else {
				return bestParent, fmt.Errorf("namespace property %s is not a resource id (was: %T)", p.Details().Path, v)
			}
		}
	}

	glue, err := knowledgebase.Downstream(
		sol.DataflowGraph(),
		sol.KnowledgeBase(),
		id,
		knowledgebase.ResourceLocalLayer,
	)
	if err != nil {
		return
	}
	pather, err := construct.ShortestPaths(sol.DataflowGraph(), id, construct.DontSkipEdges)
	if err != nil {
		return
	}
	bestParentWeight := math.MaxInt32
	var errs error
candidateLoop:
	for _, candidate := range glue {
		if GetResourceVizTag(e.Kb, view, candidate) != ParentIconTag {
			continue
		}
		path, err := pather.ShortestPath(candidate)
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
		weight, err := graph_addons.PathWeight(sol.DataflowGraph(), graph_addons.Path[construct.ResourceId](path))
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if weight < bestParentWeight {
			bestParentWeight = weight
			bestParent = candidate
		}
	}
	err = errs
	return
}

func visPaths(sol solution_context.SolutionContext, view View, source, target construct.ResourceId) ([]construct.Path, error) {
	srcTemplate, err := sol.KnowledgeBase().GetResourceTemplate(source)
	if err != nil || srcTemplate == nil {
		return nil, fmt.Errorf("has path could not find source resource %s: %w", source, err)
	}
	targetTemplate, err := sol.KnowledgeBase().GetResourceTemplate(target)
	if err != nil || targetTemplate == nil {
		return nil, fmt.Errorf("has path could not find target resource %s: %w", target, err)
	}
	if len(targetTemplate.PathSatisfaction.AsTarget) == 0 || len(srcTemplate.PathSatisfaction.AsSource) == 0 {
		return nil, nil
	}
	sourceRes, err := sol.RawView().Vertex(source)
	if err != nil {
		return nil, fmt.Errorf("has path could not find source resource %s: %w", source, err)
	}
	targetRes, err := sol.RawView().Vertex(target)
	if err != nil {
		return nil, fmt.Errorf("has path could not find target resource %s: %w", target, err)
	}

	consumed, err := knowledgebase.HasConsumedFromResource(
		sourceRes,
		targetRes,
		solution_context.DynamicCtx(sol),
	)
	if err != nil {
		return nil, err
	}
	if !consumed {
		return nil, nil
	}
	return checkPaths(sol, view, source, target)
}

func checkPaths(sol solution_context.SolutionContext, view View, source, target construct.ResourceId) ([]construct.Path, error) {
	paths, err := path_selection.GetPaths(
		sol,
		source,
		target,
		func(source, target construct.ResourceId, path construct.Path) bool {
			for _, res := range path[1 : len(path)-1] {
				switch GetResourceVizTag(sol.KnowledgeBase(), view, res) {
				case BigIconTag, ParentIconTag:
					// Don't consider paths that go through big/parent icons
					return false
				}
			}
			return true
		},
		false,
	)
	return paths, err
}
