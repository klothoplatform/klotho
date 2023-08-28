package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const KLOTHO_ID_LABEL = "klothoId"

type (
	ManifestFile interface {
		construct.Resource
		GetObject() v1.Object
		Kind() string
		Path() string
	}
	Manifest struct {
		Name            string
		ConstructRefs   construct.BaseConstructSet `yaml:"-"`
		FilePath        string
		Content         []byte
		Transformations map[string]construct.IaCValue
		Cluster         construct.ResourceId
	}

	ManifestWithValues interface {
		ManifestFile
		GetValues() map[string]construct.IaCValue
	}
)

const MANIFEST_TYPE = "manifest"

func OutputObjectAsYaml(manifest ManifestFile) (*io.RawFile, error) {
	output, err := yaml.Marshal(manifest.GetObject())
	if err != nil {
		return nil, err
	}
	return &io.RawFile{
		Content: output,
		FPath:   manifest.Path(),
	}, nil
}

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (manifest *Manifest) BaseConstructRefs() construct.BaseConstructSet {
	return manifest.ConstructRefs
}

func (manifest *Manifest) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "kubernetes",
		Type:     MANIFEST_TYPE,
		Name:     manifest.Name,
	}
}

func (manifest *Manifest) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (manifest *Manifest) GetOutputFiles() []io.File {
	if len(manifest.Content) > 0 {
		return []io.File{&io.RawFile{
			Content: manifest.Content,
			FPath:   manifest.FilePath,
		}}
	}
	return []io.File{}
}

func SetDefaultObjectMeta(resource construct.Resource, meta v1.Object) {
	meta.SetName(kubernetes.MetadataNameSanitizer.Apply(resource.Id().Name))
	if meta.GetLabels() == nil {
		meta.SetLabels(make(map[string]string))
	}
	labels := meta.GetLabels()
	labels[KLOTHO_ID_LABEL] = kubernetes.LabelValueSanitizer.Apply(resource.Id().String())
	meta.SetLabels(labels)
}

func ManifestFilePath(file ManifestFile) string {
	return fmt.Sprintf("%s_%s.yaml", file.Id().Type, file.Id().Name)
}

func KlothoIdSelector(object v1.Object) map[string]string {
	labels := object.GetLabels()
	if labels == nil {
		return map[string]string{KLOTHO_ID_LABEL: ""}
	}
	return map[string]string{KLOTHO_ID_LABEL: labels[KLOTHO_ID_LABEL]}
}
