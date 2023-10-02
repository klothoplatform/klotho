package kubernetes

import (
	"errors"
	"path"

	"github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/io"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
)

func AddCharts(g construct2.Graph) ([]io.File, error) {
	resources, err := construct2.ReverseTopologicalSort(g)
	if err != nil {
		return nil, err
	}
	var errs error
	var files []io.File
	for _, r := range resources {
		switch r.QualifiedTypeName() {
		case "aws:eks_cluster":
			charts, err := AddClusterCharts(g, r)
			errs = errors.Join(errs, err)
			files = append(files, charts...)
		}
	}
	return files, err
}

func AddClusterCharts(g construct2.Graph, cluster construct2.ResourceId) ([]io.File, error) {
	upstreams, err := construct2.DirectUpstreamDependencies(g, cluster)
	if err != nil {
		return nil, err
	}

	var helmTemplates, charts, addons, manifests []construct2.ResourceId
	for _, dep := range upstreams {
		switch dep.QualifiedTypeName() {
		case "kubernetes:manifest_file":
			helmTemplates = append(helmTemplates, dep)

		case "kubernetes:helm_chart":
			charts = append(charts, dep)

		case "aws:eks_addon":
			addons = append(addons, dep)

		case "kubernetes:manifest":
			manifests = append(manifests, dep)
		}
	}

	var knownPrereqs []construct2.ResourceId
	knownPrereqs = append(knownPrereqs, charts...)
	knownPrereqs = append(knownPrereqs, addons...)
	knownPrereqs = append(knownPrereqs, manifests...)

	prereqDownstreams := make(map[construct2.ResourceId]struct{})
	for _, prereq := range knownPrereqs {
		prereqDownstream, err := construct2.DirectDownstreamDependencies(g, prereq)
		if err != nil {
			return nil, err
		}
		for _, downstream := range prereqDownstream {
			switch downstream.QualifiedTypeName() {
			case "kubernetes:manifest_file":
				prereqDownstreams[downstream] = struct{}{}
			}
		}
	}

	var prereqsHelmTemplates, appHelmTemplates []construct2.ResourceId

	for _, object := range helmTemplates {
		if _, ok := prereqDownstreams[object]; ok {
			prereqsHelmTemplates = append(prereqsHelmTemplates, object)
		} else {
			appHelmTemplates = append(appHelmTemplates, object)
		}
	}

	gb := construct2.NewGraphBatch(g)

	var outputFiles []io.File

	if len(prereqsHelmTemplates) > 0 {
		prereqsChart, files, err := AddChart(g, cluster, prereqsHelmTemplates, "prereqs")
		if err != nil {
			return nil, err
		}
		outputFiles = append(outputFiles, files...)
		for _, prereq := range knownPrereqs {
			gb.AddEdges(construct2.Edge{Source: prereq, Target: prereqsChart.ID})
		}
	}

	if len(appHelmTemplates) == 0 {
		return outputFiles, nil
	}

	appChart, files, err := AddChart(g, cluster, appHelmTemplates, "app")
	if err != nil {
		return nil, err
	}
	outputFiles = append(outputFiles, files...)

	manifestIds := make(map[construct2.ResourceId]struct{}, len(manifests))
	for _, manifest := range manifests {
		manifestIds[manifest] = struct{}{}
	}

	var preRequisiteCharts []construct2.ResourceId
preExistingCharts:
	for _, chart := range charts {
		chartDownstream, err := construct2.DirectDownstreamDependencies(g, chart)
		if err != nil {
			return nil, err
		}
		for _, manifest := range chartDownstream {
			if manifest.QualifiedTypeName() == "kubernetes:manifest_file" {
				if _, hasManifest := manifestIds[manifest]; !hasManifest {
					preRequisiteCharts = append(preRequisiteCharts, chart)
					continue preExistingCharts
				}
			}
		}
		preRequisiteCharts = append(preRequisiteCharts, chart)
	}

	for _, chart := range preRequisiteCharts {
		gb.AddEdges(construct2.Edge{Source: appChart.ID, Target: chart})
	}
	for _, addon := range addons {
		gb.AddEdges(construct2.Edge{Source: appChart.ID, Target: addon})
	}
	for _, manifest := range manifests {
		gb.AddEdges(construct2.Edge{Source: appChart.ID, Target: manifest})
	}

	return outputFiles, gb.Err
}

func AddChart(
	g construct2.Graph,
	cluster construct2.ResourceId,
	templates []construct2.ResourceId,
	name string,
) (*construct2.Resource, []io.File, error) {
	chartRes := &construct2.Resource{
		ID: construct2.ResourceId{
			Provider:  "kubernetes",
			Type:      "helm_chart",
			Namespace: cluster.Name,
		},
		Properties: construct2.Properties{},
	}

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

	var manifestFiles []io.File
	templateValues := make(map[string]any)
	for _, manifest := range templates {
		manifestRes, err := g.Vertex(manifest)
		if err != nil {
			return nil, nil, err
		}
		helmValues := make(map[construct2.PropertyRef]construct2.PropertyPath)
		err = manifestRes.WalkProperties(func(path construct2.PropertyPath, err error) error {
			if ref, ok := path.Get().(construct2.PropertyRef); ok {
				helmValues[ref] = path
				path.Set("{{ .Values." + ref.Property + " }}")
			}
			return nil
		})
		manifestFile := io.RawFile{
			FPath: path.Join("charts", chartMeta.ChartFullPath(), "templates", manifest.String()+".yaml"),
		}
		// TODO marshal/template the file
		manifestFile.FPath = path.Join("charts", chartMeta.ChartFullPath(), "templates", manifest.String()+".yaml")
		if err != nil {
			return nil, nil, err
		}
		manifestFiles = append(manifestFiles, manifestFile)
	}

	chartYaml, err := yaml.Marshal(chartMeta.Metadata)
	if err != nil {
		return nil, nil, err
	}

	var outputFiles []io.File
	outputFiles = append(outputFiles,
		&io.RawFile{
			FPath:   path.Join("charts", chartMeta.ChartFullPath(), "Chart.yaml"),
			Content: chartYaml,
		})

	outputFiles = append(outputFiles, manifestFiles...)
}
