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
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.Resource
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
	if service.Cluster == nil {
		var downstreamClustersFound []core.Resource
		for _, res := range dag.GetAllDownstreamResources(service) {
			if classifier.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound = append(downstreamClustersFound, res)
			}
		}
		if len(downstreamClustersFound) == 1 {
			service.Cluster = downstreamClustersFound[0]
			dag.AddDependency(service, service.Cluster)
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("service %s has more than one cluster downstream", service.Id())
		}
		return core.NewOperationalResourceError(service, []string{string(core.Cluster)}, fmt.Errorf("service %s has no clusters to use", service.Id()))
	}
	return nil
}
