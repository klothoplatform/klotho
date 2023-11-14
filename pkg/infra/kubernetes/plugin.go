package kubernetes

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/config"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
)

type Plugin struct {
	Config *config.Application
	KB     *knowledgebase.KnowledgeBase
}

func (p Plugin) Name() string {
	return "kubernetes"
}

const HELM_CHARTS_DIR = "helm_charts"

func (p Plugin) Translate(ctx solution_context.SolutionContext) ([]kio.File, error) {
	internalCharts := make(map[string]*construct.Resource)
	customerCharts := make(map[string]*construct.Resource)
	resourcesInChart := make(map[construct.ResourceId][]construct.ResourceId)
	var files []kio.File

	err := construct.WalkGraph(ctx.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if id.Provider == "kubernetes" {
			output, err := AddObject(resource)
			if err != nil {
				return errors.Join(nerr, err)
			}
			if output == nil {
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
			if prop, err := resource.GetProperty("Internal"); err == nil && prop == true {
				internalChart, ok := internalCharts[clusterId.Name]
				if !ok {
					internalChart = construct.CreateResource(construct.ResourceId{
						Provider:  "kubernetes",
						Type:      "helm_chart",
						Namespace: clusterId.Name,
						Name:      "klotho-internals-chart",
					})
					chartDir := fmt.Sprintf("%s/%s/%s", HELM_CHARTS_DIR, internalChart.ID.Namespace, internalChart.ID.Name)
					err := internalChart.SetProperty("Directory", chartDir)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
					err = internalChart.SetProperty("Cluster", clusterId)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
					internalCharts[clusterId.Name] = internalChart
					err = ctx.RawView().AddVertex(internalChart)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
					err = ctx.RawView().AddEdge(internalChart.ID, clusterId)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
				}
				chartDir := fmt.Sprintf("%s/%s/%s", HELM_CHARTS_DIR, internalChart.ID.Namespace, internalChart.ID.Name)
				files = append(files, &kio.RawFile{
					FPath:   fmt.Sprintf("%s/templates/%s_%s.yaml", chartDir, internalChart.ID.Namespace, internalChart.ID.Name),
					Content: output.Content,
				})
				resourcesInChart[internalChart.ID] = append(resourcesInChart[internalChart.ID], resource.ID)
			} else {
				appChart, ok := customerCharts[clusterId.Name]
				if !ok {
					appChart = construct.CreateResource(construct.ResourceId{
						Provider:  "kubernetes",
						Type:      "helm_chart",
						Namespace: clusterId.Name,
						Name:      "application-chart",
					})
					chartDir := fmt.Sprintf("%s/%s/%s", HELM_CHARTS_DIR, appChart.ID.Namespace, appChart.ID.Name)
					err := appChart.SetProperty("Directory", chartDir)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
					err = appChart.SetProperty("Cluster", clusterId)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
					customerCharts[clusterId.Name] = appChart
					err = ctx.RawView().AddVertex(appChart)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
					err = ctx.RawView().AddEdge(appChart.ID, clusterId)
					if err != nil {
						nerr = errors.Join(nerr, err)
					}
				}
				chartDir := fmt.Sprintf("%s/%s/%s", HELM_CHARTS_DIR, appChart.ID.Namespace, appChart.ID.Name)
				files = append(files, &kio.RawFile{
					FPath:   fmt.Sprintf("%s/templates/%s_%s.yaml", chartDir, id.Type, id.Name),
					Content: output.Content,
				})
				resourcesInChart[appChart.ID] = append(resourcesInChart[appChart.ID], resource.ID)
			}
		}
		return nerr
	})
	var errs error
	if err != nil {
		errs = errors.Join(errs, err)
	}
	for _, res := range customerCharts {
		errs = errors.Join(errs, ReplaceResourcesInChart(ctx, resourcesInChart[res.ID], res))
		file, err := writeChartYaml(res)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			files = append(files, file)
		}
	}
	for _, res := range internalCharts {
		err = res.SetProperty("Internal", true)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		errs = errors.Join(errs, ReplaceResourcesInChart(ctx, resourcesInChart[res.ID], res))
		file, err := writeChartYaml(res)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			files = append(files, file)
		}
	}

	return files, errs
}

func ReplaceResourcesInChart(ctx solution_context.SolutionContext, resources []construct.ResourceId, chart *construct.Resource) error {
	var errs error
	for _, res := range resources {
		edges, err := ctx.RawView().Edges()
		if err != nil {
			return err
		}

		for _, e := range edges {
			if e.Source != res && e.Target != res {
				continue
			}
			// If the dependency is with the chart, remove it so that we dont end up depending on ourselves
			if e.Source == chart.ID || e.Target == chart.ID {
				errs = errors.Join(errs, ctx.RawView().RemoveEdge(e.Source, e.Target))
				continue
			}

			newEdge := e
			if e.Source == res {
				newEdge.Source = chart.ID
			}
			if e.Target == res {
				newEdge.Target = chart.ID
			}
			errs = errors.Join(errs,
				ctx.RawView().RemoveEdge(e.Source, e.Target),
				ctx.RawView().AddEdge(newEdge.Source, newEdge.Target, func(ep *graph.EdgeProperties) { *ep = e.Properties }),
			)
		}
		if errs != nil {
			return errs
		}
		errs = errors.Join(errs, ctx.RawView().RemoveVertex(res))
	}
	return errs
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
