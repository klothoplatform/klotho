package iac2

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	aws_resources "github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	k8s_resources "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	"helm.sh/helm/v3/pkg/chart"
	"path"
	"sigs.k8s.io/yaml"
	"strings"
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
		manifests := core.GetAllUpstreamResourcesOfType[k8s_resources.ManifestFile](dag, cluster)
		if len(manifests) == 0 {
			continue
		}

		chartContent := &chart.Chart{
			Metadata: &chart.Metadata{
				Name:        strings.ToLower(cluster.Name),
				APIVersion:  "v2",
				AppVersion:  "0.0.1",
				Version:     "0.0.1",
				KubeVersion: ">= 1.19.0-0",
				Type:        "application",
			},
		}

		var manifestFiles []core.File
		templateValues := make(map[string]any)
		for _, manifest := range manifests {
			if manifest, ok := manifest.(k8s_resources.ManifestWithValues); ok {
				for k, v := range manifest.Values() {
					templateValues[k] = v
				}
			}
			manifestFile, err := k8s_resources.OutputObjectAsYaml(manifest)
			if err != nil {
				return nil, err
			}
			manifestFiles = append(manifestFiles, manifestFile)
		}

		chartYaml, err := yaml.Marshal(chartContent.Metadata)
		if err != nil {
			return nil, err
		}

		outputFiles = append(outputFiles,
			&core.RawFile{
				FPath:   path.Join("charts", chartContent.ChartPath(), "Chart.yaml"),
				Content: chartYaml,
			})

		outputFiles = append(outputFiles, manifestFiles...)

		clusterChart := &k8s_resources.HelmChart{
			Name:          fmt.Sprintf("%s-chart", strings.ToLower(cluster.Name)),
			Directory:     path.Join("charts", chartContent.ChartPath()),
			ConstructRefs: cluster.ConstructRefs.Clone(),
			Cluster:       cluster.Id(),
			IsInternal:    true,
			Values:        templateValues,
		}
		dag.AddDependenciesReflect(clusterChart)

		// Add cluster chart dependency to all prerequisite downstream charts (i.e. charts that don't depend on any manifests contained in this chart)
		manifestIds := make(map[string]bool)
		for _, manifest := range manifests {
			manifestIds[manifest.Id().String()] = true
		}
		var preExistingCharts []*k8s_resources.HelmChart
		var clusterAddons []*aws_resources.EksAddon
		var clusterManifests []*k8s_resources.Manifest
		clusterDownstreams := dag.GetDownstreamResources(cluster)
		for _, downstream := range clusterDownstreams {
			switch res := downstream.(type) {
			case *k8s_resources.HelmChart:
				preExistingCharts = append(preExistingCharts, res)
			case *aws_resources.EksAddon:
				clusterAddons = append(clusterAddons, res)
			case *k8s_resources.Manifest:
				clusterManifests = append(clusterManifests, res)
			}
		}
		var preRequisiteCharts []*k8s_resources.HelmChart
		for _, preExistingChart := range preExistingCharts {
			downstream := core.GetAllDownstreamResourcesOfType[k8s_resources.ManifestFile](dag, preExistingChart)
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
			dag.AddDependency(clusterChart, preRequisiteChart)
		}
		for _, clusterAddon := range clusterAddons {
			dag.AddDependency(clusterChart, clusterAddon)
		}
		for _, clusterManifest := range clusterManifests {
			dag.AddDependency(clusterChart, clusterManifest)
		}
	}

	return outputFiles, nil
}
