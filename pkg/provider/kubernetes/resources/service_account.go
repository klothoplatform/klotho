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
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.IaCValue
	}
)

const (
	SERVICE_ACCOUNT_TYPE = "service_account"
)

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (sa *ServiceAccount) BaseConstructsRef() core.BaseConstructSet { return sa.ConstructRefs }

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
