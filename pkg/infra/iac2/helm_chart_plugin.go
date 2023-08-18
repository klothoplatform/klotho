package iac2

import (
	"fmt"
	"path"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	aws_resources "github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	k8s_resources "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

type (
	ChartPlugin struct {
		Config *config.Application
	}
)

func (p ChartPlugin) Name() string {
	return "helm-chart"
}

func (p ChartPlugin) Translate(dag *core.ResourceGraph) ([]core.File, error) {

	clusters := core.GetResources[*aws_resources.EksCluster](dag)
	var outputFiles []core.File

	for _, cluster := range clusters {
		var helmTemplates []k8s_resources.ManifestFile
		var preExistingCharts []*k8s_resources.HelmChart
		var clusterAddons []*aws_resources.EksAddon
		var clusterManifests []*k8s_resources.Manifest
		clusterUpstreams := dag.GetAllUpstreamResources(cluster)
		for _, upstream := range clusterUpstreams {
			switch res := upstream.(type) {
			case k8s_resources.ManifestFile:
				helmTemplates = append(helmTemplates, res)
			case *k8s_resources.HelmChart:
				preExistingCharts = append(preExistingCharts, res)
			case *aws_resources.EksAddon:
				clusterAddons = append(clusterAddons, res)
			case *k8s_resources.Manifest:
				clusterManifests = append(clusterManifests, res)
			}
		}

		var knownPrereqs []core.Resource
		for _, chart := range preExistingCharts {
			knownPrereqs = append(knownPrereqs, chart)
		}
		for _, addon := range clusterAddons {
			knownPrereqs = append(knownPrereqs, addon)
		}
		for _, manifest := range clusterManifests {
			knownPrereqs = append(knownPrereqs, manifest)
		}

		prereqDownstreams := make(map[core.Resource]bool)
		for _, prereq := range knownPrereqs {
			for _, downstream := range core.GetDownstreamResourcesOfType[k8s_resources.ManifestFile](dag, prereq) {
				prereqDownstreams[downstream] = true
			}
		}

		var prereqsHelmTemplates []k8s_resources.ManifestFile
		var appHelmTemplates []k8s_resources.ManifestFile

		for _, object := range helmTemplates {
			if _, ok := prereqDownstreams[object]; ok {
				prereqsHelmTemplates = append(prereqsHelmTemplates, object)
			} else {
				appHelmTemplates = append(appHelmTemplates, object)
			}
		}

		var err error
		var prereqsChart *k8s_resources.HelmChart
		if len(prereqsHelmTemplates) > 0 {
			prereqsChart, outputFiles, err = createChart(fmt.Sprintf("%s-prereqs", cluster.Name), cluster.Id(), dag, prereqsHelmTemplates)
			if err != nil {
				return nil, err
			}

			for _, prereq := range knownPrereqs {
				dag.AddDependency(prereq, prereqsChart)
			}
		}

		if len(appHelmTemplates) == 0 {
			continue
		}

		// Create application chart
		applicationChart, appChartOutputs, err := createChart(fmt.Sprintf("%s-application", cluster.Name), cluster.Id(), dag, appHelmTemplates)
		if err != nil {
			return nil, err
		}
		outputFiles = append(outputFiles, appChartOutputs...)

		// Add cluster chart dependency to all prerequisite upstream charts (i.e. charts that don't depend on any manifests contained in this chart)
		manifestIds := make(map[string]bool)
		for _, manifest := range helmTemplates {
			manifestIds[manifest.Id().String()] = true
		}

		var preRequisiteCharts []*k8s_resources.HelmChart
		for _, preExistingChart := range preExistingCharts {
			downstream := core.GetDownstreamResourcesOfType[k8s_resources.ManifestFile](dag, preExistingChart)
			if len(downstream) == 0 {
				preRequisiteCharts = append(preRequisiteCharts, preExistingChart)
				continue
			}
			for _, manifest := range downstream {
				if !manifestIds[manifest.Id().String()] {
					preRequisiteCharts = append(preRequisiteCharts, preExistingChart)
					continue
				}
			}
		}
		for _, preRequisiteChart := range preRequisiteCharts {
			dag.AddDependency(applicationChart, preRequisiteChart)
		}
		for _, clusterAddon := range clusterAddons {
			dag.AddDependency(applicationChart, clusterAddon)
		}
		for _, clusterManifest := range clusterManifests {
			dag.AddDependency(applicationChart, clusterManifest)
		}
	}

	return outputFiles, nil
}

func createChart(name string, cluster core.ResourceId, dag *core.ResourceGraph, templates []k8s_resources.ManifestFile) (*k8s_resources.HelmChart, []core.File, error) {
	chartMeta := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:        name,
			APIVersion:  "v2",
			AppVersion:  "0.0.1",
			Version:     "0.0.1",
			KubeVersion: ">= 1.22.0-0",
			Type:        "application",
		},
	}

	var manifestFiles []core.File
	templateValues := make(map[string]any)
	for _, manifest := range templates {
		if manifest, ok := manifest.(k8s_resources.ManifestWithValues); ok {
			for k, v := range manifest.GetValues() {
				templateValues[k] = v
			}
		}
		manifestFile, err := k8s_resources.OutputObjectAsYaml(manifest)
		manifestFile.FPath = path.Join("charts", chartMeta.ChartFullPath(), "templates", manifest.Id().String()+".yaml")
		if err != nil {
			return nil, nil, err
		}
		manifestFiles = append(manifestFiles, manifestFile)
	}

	chartYaml, err := yaml.Marshal(chartMeta.Metadata)
	if err != nil {
		return nil, nil, err
	}

	var outputFiles []core.File
	outputFiles = append(outputFiles,
		&core.RawFile{
			FPath:   path.Join("charts", chartMeta.ChartFullPath(), "Chart.yaml"),
			Content: chartYaml,
		})

	outputFiles = append(outputFiles, manifestFiles...)

	helmChart := &k8s_resources.HelmChart{
		Name:          fmt.Sprintf("%s-%s-chart", strings.ToLower(cluster.Name), name),
		Directory:     path.Join("charts", chartMeta.ChartFullPath()),
		ConstructRefs: core.BaseConstructSetOf(),
		Cluster:       cluster,
		IsInternal:    true,
		Values:        templateValues,
	}

	for _, template := range templates {
		helmChart.ConstructRefs.Add(template)
	}

	dag.AddDependenciesReflect(helmChart)
	return helmChart, outputFiles, nil
}
