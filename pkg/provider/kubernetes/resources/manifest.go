package resources

import (
	"fmt"
	"path"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const KLOTHO_ID_LABEL = "klothoId"

type (
	ManifestFile interface {
		core.Resource
		GetObject() runtime.Object
		Kind() string
		Path() string
	}
	Manifest struct {
		Name            string
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		FilePath        string
		Content         []byte
		Transformations map[string]core.IaCValue
		Cluster         core.ResourceId
	}

	ManifestWithValues interface {
		ManifestFile
		Values() map[string]core.IaCValue
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

func (manifest *Manifest) GetOutputFiles() []core.File {
	if len(manifest.Content) > 0 {
		return []core.File{&core.RawFile{
			Content: manifest.Content,
			FPath:   manifest.FilePath,
		}}
	}
	return []core.File{}
}

func SetDefaultObjectMeta(resource core.Resource, meta v1.Object) {
	meta.SetName(kubernetes.MetadataNameSanitizer.Apply(resource.Id().Name))
	if meta.GetLabels() == nil {
		meta.SetLabels(make(map[string]string))
	}
	labels := meta.GetLabels()
	labels[KLOTHO_ID_LABEL] = kubernetes.LabelValueSanitizer.Apply(resource.Id().String())
	meta.SetLabels(labels)
}

func ManifestFilePath(file ManifestFile, clusterId core.ResourceId) string {
	return path.Join("charts", clusterId.Name, "templates", fmt.Sprintf("%s_%s.yaml", file.Id().Type, file.Id().Name))
}

func KlothoIdSelector(object v1.Object) map[string]string {
	labels := object.GetLabels()
	if labels == nil {
		return map[string]string{KLOTHO_ID_LABEL: ""}
	}
	return map[string]string{KLOTHO_ID_LABEL: labels[KLOTHO_ID_LABEL]}
}
