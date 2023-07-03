package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
)

type (
	Namespace struct {
		ConstructRefs   core.BaseConstructSet
		Object          corev1.Namespace
		Transformations map[string]core.IaCValue
		FilePath        string
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
		Name:     namespace.Object.Name,
	}
}

func (namespace *Namespace) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}

func (namespace *Namespace) Kind() string {
	return namespace.Object.Kind
}

func (namespace *Namespace) Path() string {
	return namespace.FilePath
}
