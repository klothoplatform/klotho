package resources

import (
	"bytes"
	"fmt"
	"path"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
)

const HELM_CHART_TYPE = "helm_chart"

type HelmChart struct {
	Name      string
	Chart     string
	Directory string
	Files     []ManifestFile

	ConstructRefs core.BaseConstructSet
	Cluster       core.ResourceId
	Repo          string
	Version       string
	Namespace     string
	Values        map[string]any
	// IsInternal is a flag used to identify charts as being included in application by Klotho itself
	IsInternal bool
}

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (chart *HelmChart) BaseConstructRefs() core.BaseConstructSet { return chart.ConstructRefs }

func (chart *HelmChart) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "kubernetes",
		Type:     HELM_CHART_TYPE,
		Name:     chart.Name,
	}
}

func (chart *HelmChart) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (t *HelmChart) GetOutputFiles() []core.File {
	var outputFiles []core.File
	for _, file := range t.Files {
		buf := &bytes.Buffer{}
		manifestFile, err := OutputObjectAsYaml(file)
		if err != nil {
			panic(err)
		}
		_, err = manifestFile.WriteTo(buf)
		if err != nil {
			panic(err)
		}
		outputFiles = append(outputFiles, &core.RawFile{
			FPath:   path.Join("charts", file.Path()),
			Content: buf.Bytes(),
		})
	}
	return outputFiles
}

func (chart *HelmChart) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if chart.Cluster.IsZero() {
		var downstreamClustersFound []core.Resource
		for _, res := range dag.GetAllDownstreamResources(chart) {
			if classifier.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound = append(downstreamClustersFound, res)
			}
		}
		if len(downstreamClustersFound) == 1 {
			chart.Cluster = downstreamClustersFound[0].Id()
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("helm chart %s has more than one cluster downstream", chart.Id())
		}

		return core.NewOperationalResourceError(chart, []string{string(core.Cluster)}, fmt.Errorf("helm chart %s has no clusters to use", chart.Id()))
	}
	return nil
}
