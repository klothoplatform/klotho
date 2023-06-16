package resources

import (
	"bytes"
	"path"

	"github.com/klothoplatform/klotho/pkg/core"
)

const HELM_CHART_TYPE = "helm_chart"

type HelmChart struct {
	Name      string
	Chart     string
	Directory string
	Files     []Manifest

	ConstructRefs    core.BaseConstructSet
	ClustersProvider core.IaCValue
	Repo             string
	Version          string
	Namespace        string
	Values           map[string]any
}

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (chart *HelmChart) BaseConstructsRef() core.BaseConstructSet { return chart.ConstructRefs }

func (chart *HelmChart) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "kubernetes",
		Type:     HELM_CHART_TYPE,
		Name:     chart.Name,
	}
}
func (t *HelmChart) GetOutputFiles() []core.File {
	var outputFiles []core.File
	for _, file := range t.Files {
		buf := &bytes.Buffer{}
		_, err := file.OutputYAML().WriteTo(buf)
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
