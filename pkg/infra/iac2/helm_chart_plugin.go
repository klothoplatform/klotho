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
	}

	return outputFiles, nil
}
