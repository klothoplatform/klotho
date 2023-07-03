package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	Namespace struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Namespace
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.IaCValue
	}
)

const (
	NAMESPACE_TYPE = "namespace"
)

func (namespace *Namespace) BaseConstructsRef() core.BaseConstructSet {
	return namespace.ConstructRefs
}

func (namespace *Namespace) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     NAMESPACE_TYPE,
		Name:     namespace.Name,
	}
}

func (namespace *Namespace) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (namespace *Namespace) GetObject() runtime.Object {
	return namespace.Object
}

func (namespace *Namespace) Kind() string {
	return namespace.Object.Kind
}

func (namespace *Namespace) Path() string {
	return namespace.FilePath
}
