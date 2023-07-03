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
func (k *KubernetesProvider) ExpandConstruct(construct core.Construct, cg *core.ConstructGraph, dag *core.ResourceGraph, constructType string, attributes map[string]any) (directlyMappedResources []core.Resource, err error) {
	return nil, nil
}
