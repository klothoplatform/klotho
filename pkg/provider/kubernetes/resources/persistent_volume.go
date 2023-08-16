package resources

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PERSISTENT_VOLUME_TYPE = "persistent_volume"
)

type (
	PersistentVolume struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Object        *corev1.PersistentVolume
		Values        map[string]core.IaCValue
		FilePath      string
		Cluster       core.ResourceId
	}
)

func (pv *PersistentVolume) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     PERSISTENT_VOLUME_TYPE,
		Name:     pv.Name,
	}
}

func (pv *PersistentVolume) BaseConstructRefs() core.BaseConstructSet {
	return pv.ConstructRefs
}

func (pv *PersistentVolume) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (pv *PersistentVolume) GetObject() v1.Object {
	return pv.Object
}

func (pv *PersistentVolume) Kind() string {
	return pv.Object.Kind
}

func (pv *PersistentVolume) Path() string {
	return pv.FilePath
}

type PersistentVolumeCreateParams struct {
	Name          string
	ConstructRefs core.BaseConstructSet
}

func (pv *PersistentVolume) Create(dag *core.ResourceGraph, params PersistentVolumeCreateParams) error {
	pv.Name = fmt.Sprintf("%s-pv", params.Name)
	pv.ConstructRefs = params.ConstructRefs.Clone()
	pv.Object = &corev1.PersistentVolume{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolume",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: kubernetes.RFC1035LabelSanitizer.Apply(pv.Name),
		},
	}
	return nil
}

func (pv *PersistentVolume) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if pv.Cluster.IsZero() {
		return fmt.Errorf("%s has no cluster", pv.Id())
	}
	SetDefaultObjectMeta(pv, pv.Object.GetObjectMeta())
	pv.FilePath = ManifestFilePath(pv)
	return nil
}

func (pv *PersistentVolume) GetValues() map[string]core.IaCValue {
	return pv.Values
}
