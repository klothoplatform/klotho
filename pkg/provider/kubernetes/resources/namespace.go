package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
)

type (
	Namespace struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          corev1.Namespace
		Transformations map[string]core.IaCValue
		FilePath        string
	}
)

const (
	NAMESPACE_TYPE = "namespace"
)

func (deployment *Namespace) BaseConstructsRef() core.BaseConstructSet {
	return deployment.ConstructRefs
}

func (deployment *Namespace) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     DEPLOYMENT_TYPE,
		Name:     deployment.Name,
	}
}

func (deployment *Namespace) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}
