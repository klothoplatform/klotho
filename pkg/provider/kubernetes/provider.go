package kubernetes

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

type (
	KubernetesProvider struct {
		AppName string
	}
)

func (k *KubernetesProvider) Name() string {
	return "kubernetes"
}
func (k *KubernetesProvider) ListResources() []core.Resource {
	return resources.ListAll()
}
func (k *KubernetesProvider) CreateResourceFromId(id core.ResourceId, dag *core.ConstructGraph) (core.Resource, error) {
	return nil, nil
}
