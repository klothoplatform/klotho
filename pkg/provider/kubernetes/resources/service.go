package resources

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	Service struct {
		Name            string
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		Object          *corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.ResourceId
	}
)

const (
	SERVICE_TYPE = "service"
)

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (service *Service) BaseConstructRefs() core.BaseConstructSet { return service.ConstructRefs }

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

func (service *Service) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if service.Cluster.Name == "" {
		return fmt.Errorf("service %s has no cluster", service.Name)
	}

	SetDefaultObjectMeta(service, service.Object.GetObjectMeta())
	service.FilePath = ManifestFilePath(service, service.Cluster)
	return nil
}

func (service *Service) Values() map[string]core.IaCValue {
	return service.Transformations
}
