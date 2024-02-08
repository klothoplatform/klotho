package kubernetes

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
)

type Plugin struct {
	AppName          string
	KB               *knowledgebase.KnowledgeBase
	files            []kio.File
	resourcesInChart map[construct.ResourceId][]construct.ResourceId
}

func (p Plugin) Name() string {
	return "kubernetes"
}

const HELM_CHARTS_DIR = "helm_charts"

func (p Plugin) Translate(ctx solution_context.SolutionContext) ([]kio.File, error) {
	internalCharts := make(map[string]*construct.Resource)
	customerCharts := make(map[string]*construct.Resource)
	p.resourcesInChart = make(map[construct.ResourceId][]construct.ResourceId)

	err := construct.WalkGraphReverse(ctx.DeploymentGraph(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if id.Provider == "kubernetes" {
			if !includeObjectInChart(id) {
				return nerr
			}
			cluster, err := resource.GetProperty("Cluster")
			if err != nil {
				return errors.Join(nerr, err)
			}
			clusterId, ok := cluster.(construct.ResourceId)
			if !ok {
				return errors.Join(nerr, fmt.Errorf("cluster property is not a resource id"))
			}

			// attempt to add to internal chart
			internalChart, ok := internalCharts[clusterId.Name]
			if !ok {
				internalChart, err = p.createChart("internal-chart", clusterId, ctx)
				if err != nil {
					return errors.Join(nerr, err)
				}
				internalCharts[clusterId.Name] = internalChart
			}
			placed, err := p.placeResourceInChart(ctx, resource, internalChart)
			if err != nil {
				return err
			}

			if !placed {
				// attempt to add to app chart for cluster if it cannot be in the internal chart
				appChart, ok := customerCharts[clusterId.Name]
				if !ok {

					appChart, err = p.createChart("application-chart", clusterId, ctx)
					if err != nil {
						return errors.Join(nerr, err)
					}

					customerCharts[clusterId.Name] = appChart

				}
				placed, err = p.placeResourceInChart(ctx, resource, appChart)
				if err != nil {
					return errors.Join(nerr, err)
				}
				if !placed {
					return errors.Join(nerr, fmt.Errorf("could not place resource %s in chart", resource.ID))
				}
			}

		}
		return nerr
	})
	return p.files, err
}

func (p *Plugin) placeResourceInChart(ctx solution_context.SolutionContext, resource *construct.Resource, chart *construct.Resource) (
	bool,
	error,
) {
	edges, err := ctx.DeploymentGraph().Edges()
	if err != nil {
		return false, err
	}
	edgesToRemove := make([]graph.Edge[construct.ResourceId], 0)
	edgesToAdd := make([]graph.Edge[construct.ResourceId], 0)
	tmpGraph, err := ctx.DeploymentGraph().Clone()
	if err != nil {
		return false, err
	}
	for _, e := range edges {
		if e.Source != resource.ID && e.Target != resource.ID {
			continue
		}
		newEdge := e
		if e.Source == resource.ID {
			newEdge.Source = chart.ID
		}
		if e.Target == resource.ID {
			newEdge.Target = chart.ID
		}
		err = tmpGraph.RemoveEdge(e.Source, e.Target)
		if err != nil {
			return false, err
		}
		edgesToRemove = append(edgesToRemove, e)
		if newEdge.Source == newEdge.Target {
			continue
		}
		err = tmpGraph.AddEdge(newEdge.Source, newEdge.Target)
		switch {
		case errors.Is(err, graph.ErrEdgeCreatesCycle):
			return false, nil
		}
		edgesToAdd = append(edgesToAdd, newEdge)
	}
	for _, e := range edgesToRemove {
		err = ctx.DeploymentGraph().RemoveEdge(e.Source, e.Target)
		if err != nil {
			return false, err
		}
	}
	for _, e := range edgesToAdd {
		err = ctx.DeploymentGraph().AddEdge(e.Source, e.Target)
		if err != nil {
			return false, err
		}
	}

	err = ctx.DeploymentGraph().RemoveVertex(resource.ID)
	if err != nil {
		return false, fmt.Errorf("could not remove vertex %s from graph: %s", resource.ID, err)
	}
	chartDir, err := chart.GetProperty("Directory")
	if err != nil {
		return false, err
	}
	output, err := AddObject(resource)
	if err != nil {
		return false, err
	}
	if output == nil {
		return false, err
	}
	p.resourcesInChart[chart.ID] = append(p.resourcesInChart[chart.ID], resource.ID)
	p.files = append(p.files, &kio.RawFile{
		FPath:   fmt.Sprintf("%s/templates/%s_%s.yaml", chartDir, resource.ID.Type, resource.ID.Name),
		Content: output.Content,
	})
	err = chart.AppendProperty("Values", output.Values)
	if err != nil {
		return true, err
	}
	return true, nil
}

func writeChartYaml(c *construct.Resource) (kio.File, error) {
	chartContent := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        c.ID.Name,
			APIVersion:  "v2",
			AppVersion:  "0.0.1",
			Version:     "0.0.1",
			KubeVersion: ">= 1.19.0-0",
			Type:        "application",
		},
	}
	output, err := yaml.Marshal(chartContent.Metadata)
	if err != nil {
		return nil, err
	}
	directory, err := c.GetProperty("Directory")
	if err != nil {
		return nil, err
	}
	return &kio.RawFile{
		FPath:   fmt.Sprintf("%s/Chart.yaml", directory),
		Content: output,
	}, nil
}

func (p *Plugin) createChart(name string, clusterId construct.ResourceId, ctx solution_context.SolutionContext) (*construct.Resource, error) {
	chart, err := knowledgebase.CreateResource(ctx.KnowledgeBase(), construct.ResourceId{
		Provider:  "kubernetes",
		Type:      "helm_chart",
		Namespace: clusterId.Name,
		Name:      name,
	})
	if err != nil {
		return chart, err
	}
	chartDir := fmt.Sprintf("%s/%s/%s", HELM_CHARTS_DIR, chart.ID.Namespace, chart.ID.Name)
	err = chart.SetProperty("Directory", chartDir)
	if err != nil {
		return chart, err
	}
	err = chart.SetProperty("Cluster", clusterId)
	if err != nil {
		return chart, err
	}
	err = ctx.RawView().AddVertex(chart)
	if err != nil {
		return chart, err
	}
	err = ctx.RawView().AddEdge(chart.ID, clusterId)
	if err != nil {
		return chart, err
	}
	file, err := writeChartYaml(chart)
	if err != nil {
		return chart, err
	} else {
		p.files = append(p.files, file)
	}
	return chart, nil
}
