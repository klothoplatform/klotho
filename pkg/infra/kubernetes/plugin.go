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
	internalChart := construct.CreateResource(construct.ResourceId{Provider: "kubernetes", Type: "helm_chart", Name: "klotho-internals-chart"})
	customerChart := construct.CreateResource(construct.ResourceId{Provider: "kubernetes", Type: "helm_chart", Name: "application-chart"})

	files := make([]kio.File, 0)
	resourcesInInternalChart := make(map[construct.ResourceId]bool)
	resourcesInCustomerChart := make(map[construct.ResourceId]bool)

	err := construct.WalkGraph(ctx.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if id.Provider == "kubernetes" {
			output, err := AddObject(resource)
			if err != nil {
				return errors.Join(nerr, err)
			}
			if output == nil {
				return nerr
			}
			if prop, err := resource.GetProperty("Internal"); err == nil && prop == true {
				files = append(files, &kio.RawFile{
					FPath:   fmt.Sprintf("%s/%s/templates/%s_%s.yaml", HELM_CHARTS_DIR, internalChart.ID.Name, id.Type, id.Name),
					Content: output.Content,
				})
				resourcesInInternalChart[id] = true
			} else {
				files = append(files, &kio.RawFile{
					FPath:   fmt.Sprintf("%s/%s/templates/%s_%s.yaml", HELM_CHARTS_DIR, customerChart.ID.Name, id.Type, id.Name),
					Content: output.Content,
				})
				resourcesInCustomerChart[id] = true
			}
		}
		return nerr
	})
	var errs error
	if err != nil {
		errs = errors.Join(errs, err)
	}
	if len(resourcesInCustomerChart) > 0 {
		errs := errors.Join(errs, ReplaceResourcesInChart(ctx, resourcesInCustomerChart, customerChart))
		file, err := writeChartYaml(customerChart)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			files = append(files, file)
		}
	}
	if len(resourcesInInternalChart) > 0 {
		err = internalChart.SetProperty("Internal", true)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		err = internalChart.SetProperty("Directory", internalChart.ID.Name)
		if err != nil {
			errs = errors.Join(errs, err)
		}
		err = internalChart.SetProperty("Directory", customerChart.ID.Name)
		if err != nil {
			errs = errors.Join(errs, err)
		}

		if err != nil {
			errs = errors.Join(errs, err)
		}
		errs = errors.Join(errs, ReplaceResourcesInChart(ctx, resourcesInInternalChart, internalChart))

		// Add the chart.yaml files
		file, err := writeChartYaml(internalChart)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			files = append(files, file)
		}
	}

	return files, errs
}

func ReplaceResourcesInChart(ctx solution_context.SolutionContext, resources map[construct.ResourceId]bool, chart *construct.Resource) error {
	var errs error
	for res := range resources {
		_, err := ctx.RawView().Vertex(chart.ID)
		switch {
		case errors.Is(err, graph.ErrVertexNotFound):
			errs = errors.Join(errs, ctx.RawView().AddVertex(chart))
		case err != nil:
			errs = errors.Join(errs, err)
			continue
		}

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
	return &kio.RawFile{
		FPath:   fmt.Sprintf("%s/%s/Chart.yaml", HELM_CHARTS_DIR, c.ID.Name),
		Content: output,
	}, nil
}
