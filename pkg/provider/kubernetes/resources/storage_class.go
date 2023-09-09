package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	v1 "k8s.io/api/storage/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	STORAGE_CLASS_TYPE = "storage_class"
)

type (
	StorageClass struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Object        *v1.StorageClass
		Values        map[string]construct.IaCValue
		FilePath      string
		Cluster       construct.ResourceId
	}
)

func (sc *StorageClass) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     STORAGE_CLASS_TYPE,
		Name:     sc.Name,
	}
}

func (sc *StorageClass) BaseConstructRefs() construct.BaseConstructSet {
	return sc.ConstructRefs
}

func (sc *StorageClass) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (sc *StorageClass) GetObject() metaV1.Object {
	return sc.Object
}

func (sc *StorageClass) Kind() string {
	return sc.Object.Kind
}

func (sc *StorageClass) Path() string {
	return sc.FilePath
}

type StorageClassCreateParams struct {
	Name          string
	ConstructRefs construct.BaseConstructSet
}

func (sc *StorageClass) Create(dag *construct.ResourceGraph, params StorageClassCreateParams) error {
	sc.Name = fmt.Sprintf("%s-sc", params.Name)
	sc.ConstructRefs = params.ConstructRefs
	sc.Object = &v1.StorageClass{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name: kubernetes.RFC1035LabelSanitizer.Apply(sc.Name),
		},
	}
	return nil
}

func (sc *StorageClass) MakeOperational(dag *construct.ResourceGraph, appName string, classifier *classification.ClassificationDocument) error {
	if sc.Cluster.IsZero() {
		return fmt.Errorf("%s has no cluster", sc.Id())
	}
	SetDefaultObjectMeta(sc, sc.Object.GetObjectMeta())
	sc.FilePath = ManifestFilePath(sc)
	return nil
}

func (sc *StorageClass) GetValues() map[string]construct.IaCValue {
	return sc.Values
}
