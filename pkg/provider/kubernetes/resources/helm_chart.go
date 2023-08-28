package resources

import (
	"bytes"
	"path"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
)

const HELM_CHART_TYPE = "helm_chart"

type HelmChart struct {
	Name          string
	Chart         string
	Directory     string
	Files         []ManifestFile
	ConstructRefs construct.BaseConstructSet `yaml:"-"`
	Cluster       construct.ResourceId
	Repo          string
	Version       string
	Namespace     string
	Values        map[string]any
	// IsInternal is a flag used to identify charts as being included in application by Klotho itself
	IsInternal bool
}

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (chart *HelmChart) BaseConstructRefs() construct.BaseConstructSet { return chart.ConstructRefs }

func (chart *HelmChart) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "kubernetes",
		Type:     HELM_CHART_TYPE,
		Name:     chart.Name,
	}
}

func (chart *HelmChart) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (chart *HelmChart) SanitizedName() string {
	return kubernetes.HelmReleaseNameSanitizer.Apply(chart.Name)
}

func (t *HelmChart) GetOutputFiles() []io.File {
	var outputFiles []io.File
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
		outputFiles = append(outputFiles, &io.RawFile{
			FPath:   path.Join("charts", file.Path()),
			Content: buf.Bytes(),
		})
	}
	return outputFiles
}
