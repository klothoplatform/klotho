package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"gopkg.in/yaml.v2"
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
		Cluster         core.Resource
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

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (manifest *Manifest) BaseConstructsRef() core.BaseConstructSet { return manifest.ConstructRefs }

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

func (manifest *Manifest) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if manifest.Cluster == nil {
		var downstreamClustersFound []core.Resource
		for _, res := range dag.GetAllDownstreamResources(manifest) {
			if core.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound = append(downstreamClustersFound, res)
			}
		}
		if len(downstreamClustersFound) == 1 {
			manifest.Cluster = downstreamClustersFound[0]
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("helm chart %s has more than one cluster downstream", manifest.Id())
		}
	}
	return core.NewOperationalResourceError(manifest, []string{string(core.Cluster)}, fmt.Errorf("helm chart %s has no clusters to use", manifest.Id()))
}
