package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	Service struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.IaCValue
	}
)

const (
	SERVICE_TYPE = "service"
)

// BaseConstructsRef returns a slice containing the ids of any Klotho constructs is correlated to
func (service *Service) BaseConstructsRef() core.BaseConstructSet { return service.ConstructRefs }

func (service *Service) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_TYPE,
		Name:     service.Name,
	}
}

func (service *Service) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (service *Service) GetObject() runtime.Object {
	return service.Object
}
func (service *Service) Kind() string {
	return service.Object.Kind
}

func (service *Service) Path() string {
	return service.FilePath
}
