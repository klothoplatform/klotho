package resources

import (
	"bytes"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	"path"
)

const HELM_CHART_TYPE = "helm_chart"

type HelmChart struct {
	Name      string
	Chart     string
	Directory string
	Files     []ManifestFile

	ConstructRefs core.BaseConstructSet `yaml:"-"`
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

func (chart *HelmChart) SanitizedName() string {
	return kubernetes.HelmReleaseNameSanitizer.Apply(chart.Name)
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
