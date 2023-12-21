package engine2

import (
	"errors"
	"fmt"
	"math"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
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

func (e *Engine) GetResourceVizTag(view View, resource construct.ResourceId) Tag {
	template, err := e.Kb.GetResourceTemplate(resource)

	if template == nil || err != nil {
		return NoRenderTag
	}
	tag, found := template.Views[string(view)]
	if !found {
		return NoRenderTag
	}
	return Tag(tag)
}

func (e *Engine) GetViewsDag(view View, ctx solution_context.SolutionContext) (visualizer.VisGraph, error) {
	viewDag := visualizer.NewVisGraph()
	var graph construct.Graph
	if view == IACView {
		graph = ctx.DeploymentGraph()
	} else {
		graph = ctx.DataflowGraph()
	}
	connGraph, err := visualizer.NewConnectionGraph(graph, e.Kb)
	if err != nil {
		return nil, fmt.Errorf("failed to construct weighted dag: %w", err)
	}
	err = errors.Join(
		construct.GraphToSVG(connGraph.Network, "net-topo"),
		construct.GraphToSVG(connGraph.Permissions, "perm-topo"),
	)
	if err != nil {
		zap.S().Errorf("failed to construct svg: %v", err)
	}

	ids, err := construct.ReverseTopologicalSort(graph)
	if err != nil {
		return nil, err
	}

	var errs error

	// First pass gets all the vertices (groups or big icons)
	for _, id := range ids {
		var err error
		switch tag := e.GetResourceVizTag(view, id); tag {
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
		switch tag := e.GetResourceVizTag(view, id); tag {
		case NoRenderTag:
			continue
		case ParentIconTag:
			continue
		case BigIconTag:
			err := e.handleBigIcon(view, connGraph, viewDag, id)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("failed to handle big icon %s: %w", id, err))
			}
		case SmallIconTag:
			err := e.handleSmallIcon(view, connGraph.Undirected, viewDag, id)
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
	view View,
	g visualizer.ConnectionGraph,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) error {
	source, err := viewDag.Vertex(id)
	if err != nil {
		return err
	}
	parent, err := e.findParent(view, g, viewDag, id)
	if err != nil {
		return err
	}
	source.Parent = parent

	err = g.ForEachTarget(id, func(target construct.ResourceId, netPath, permPath construct.Path) error {
		if target == id {
			return nil
		}
		_, inGraphErr := viewDag.Vertex(target)
		if inGraphErr != nil {
			if errors.Is(inGraphErr, graph.ErrVertexNotFound) {
				// except for small icons, we don't care about anything that's not already in the graph
				return nil
			} else {
				return inGraphErr
			}
		}
		return viewDag.AddEdge(
			id,
			target,
			graph.EdgeData(map[string]any{"netPath": netPath, "permPath": permPath}),
		)
	})
	return err
}

func (e *Engine) findParent(
	view View,
	g visualizer.ConnectionGraph,
	viewDag visualizer.VisGraph,
	id construct.ResourceId,
) (bestParent construct.ResourceId, err error) {
	ids, err := construct.TopologicalSort(viewDag)
	if err != nil {
		return
	}
	pather, err := construct.ShortestPaths(g.Undirected, id, construct.DontSkipEdges)
	if err != nil {
		return
	}
	bestParentWeight := math.MaxInt32
	var errs error
candidateLoop:
	for _, id := range ids {
		if e.GetResourceVizTag(view, id) != ParentIconTag {
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
		weight, err := graph_addons.PathWeight(g.Undirected, graph_addons.Path[construct.ResourceId](path))
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
