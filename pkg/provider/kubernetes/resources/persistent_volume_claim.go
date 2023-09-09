package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PERSISTENT_VOLUME_CLAIM_TYPE = "persistent_volume_claim"
)

type (
	PersistentVolumeClaim struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Object        *corev1.PersistentVolumeClaim
		Values        map[string]construct.IaCValue
		FilePath      string
		Cluster       construct.ResourceId
	}
)

func (pvc *PersistentVolumeClaim) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     PERSISTENT_VOLUME_CLAIM_TYPE,
		Name:     pvc.Name,
	}
}

func (pvc *PersistentVolumeClaim) BaseConstructRefs() construct.BaseConstructSet {
	return pvc.ConstructRefs
}

func (pvc *PersistentVolumeClaim) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (pvc *PersistentVolumeClaim) GetObject() v1.Object {
	return pvc.Object
}

func (pvc *PersistentVolumeClaim) Kind() string {
	return pvc.Object.Kind
}

func (pvc *PersistentVolumeClaim) Path() string {
	return pvc.FilePath
}

type PersistentVolumeClaimCreateParams struct {
	Name          string
	ConstructRefs construct.BaseConstructSet
}

func (pvc *PersistentVolumeClaim) Create(dag *construct.ResourceGraph, params PersistentVolumeCreateParams) error {
	pvc.Name = fmt.Sprintf("%s-pvc", params.Name)
	pvc.ConstructRefs = params.ConstructRefs
	pvc.Object = &corev1.PersistentVolumeClaim{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PersistentVolumeClaim",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: kubernetes.RFC1035LabelSanitizer.Apply(pvc.Name),
		},
	}

	return nil
}

func (pvc *PersistentVolumeClaim) MakeOperational(dag *construct.ResourceGraph, appName string, classifier *classification.ClassificationDocument) error {
	if pvc.Cluster.IsZero() {
		return fmt.Errorf("%s has no cluster", pvc.Id())
	}
	SetDefaultObjectMeta(pvc, pvc.Object.GetObjectMeta())
	pvc.FilePath = ManifestFilePath(pvc)
	return nil
}

func (pvc *PersistentVolumeClaim) GetValues() map[string]construct.IaCValue {
	return pvc.Values
}
