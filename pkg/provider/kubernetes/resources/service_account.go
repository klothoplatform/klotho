package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	ServiceAccount struct {
		Name            string
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		Object          *corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.ResourceId
	}
)

const (
	SERVICE_ACCOUNT_TYPE = "service_account"
)

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (sa *ServiceAccount) BaseConstructRefs() core.BaseConstructSet { return sa.ConstructRefs }

func (sa *ServiceAccount) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_ACCOUNT_TYPE,
		Name:     sa.Name,
	}
}

func (sa *ServiceAccount) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (sa *ServiceAccount) GetObject() runtime.Object {
	return sa.Object
}

func (sa *ServiceAccount) Kind() string {
	return sa.Object.Kind
}

func (sa *ServiceAccount) Path() string {
	return sa.FilePath
}

func (sa *ServiceAccount) GetResourcesUsingServiceAccount(dag *core.ResourceGraph) []core.Resource {
	var pods []core.Resource
	for _, res := range dag.GetAllUpstreamResources(sa) {
		if pod, ok := res.(*Pod); ok {
			if pod.Object.Spec.ServiceAccountName == sa.Name {
				pods = append(pods, pod)
			}
		} else if deployment, ok := res.(*Deployment); ok {
			if deployment.Object.Spec.Template.Spec.ServiceAccountName == sa.Name {
				pods = append(pods, deployment)
			}
		}
	}
	return pods
}
