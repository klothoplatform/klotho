package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	ManifestFile interface {
		core.Resource
		GetObject() runtime.Object
		Kind() string
		Path() string
	}
	Manifest struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		FilePath        string
		Content         []byte
		Transformations map[string]core.IaCValue
		Cluster         core.ResourceId
	}
)

const MANIFEST_TYPE = "manifest"

func OutputObjectAsYaml(manifest ManifestFile) (core.File, error) {
	output, err := yaml.Marshal(manifest.GetObject())
	if err != nil {
		return nil, err
	}
	return &core.RawFile{
		Content: output,
		FPath:   manifest.Path(),
	}, nil
}

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (manifest *Manifest) BaseConstructRefs() core.BaseConstructSet { return manifest.ConstructRefs }

func (manifest *Manifest) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "kubernetes",
		Type:     MANIFEST_TYPE,
		Name:     manifest.Name,
	}
}

func (manifest *Manifest) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
