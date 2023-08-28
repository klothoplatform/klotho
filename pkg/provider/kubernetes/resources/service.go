package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/sanitization/kubernetes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type (
	Service struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Object        *corev1.Service
		Values        map[string]construct.IaCValue
		FilePath      string
		Cluster       construct.ResourceId
	}
)

const (
	SERVICE_TYPE = "service"
)

// BaseConstructRefs returns a slice containing the ids of any Klotho constructs is correlated to
func (service *Service) BaseConstructRefs() construct.BaseConstructSet { return service.ConstructRefs }

func (service *Service) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_TYPE,
		Name:     service.Name,
	}
}

func (service *Service) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (service *Service) GetObject() v1.Object {
	return service.Object
}
func (service *Service) Kind() string {
	return service.Object.Kind
}

func (service *Service) Path() string {
	return service.FilePath
}

func (service *Service) MakeOperational(dag *construct.ResourceGraph, appName string, classifier classification.Classifier) error {
	if service.Cluster.Name == "" {
		return fmt.Errorf("service %s has no cluster", service.Name)
	}

	if service.Object == nil {
		service.Object = &corev1.Service{}
	}

	SetDefaultObjectMeta(service, service.Object.GetObjectMeta())
	service.FilePath = ManifestFilePath(service)

	return nil
}

func (service *Service) MapContainerPorts(parentObjectName string, containers []corev1.Container) error {
	for _, container := range containers {
		if len(container.Ports) == 0 {
			return fmt.Errorf("container %s associated with service %s has no ports", container.Name, service.Name)
		}

		currentServicePortIndexes := make(map[int32]int)
		for i, port := range service.Object.Spec.Ports {
			currentServicePortIndexes[port.Port] = i
		}

		for _, port := range container.Ports {
			servicePort := corev1.ServicePort{
				Name:       kubernetes.RFC1123LabelSanitizer.Apply(fmt.Sprintf("%s-%s-%d", parentObjectName, container.Name, port.ContainerPort)),
				Protocol:   port.Protocol,
				Port:       port.HostPort,
				TargetPort: intstr.FromInt(int(port.ContainerPort)),
			}
			if i, ok := currentServicePortIndexes[port.HostPort]; ok {
				service.Object.Spec.Ports[i] = servicePort
			} else {
				service.Object.Spec.Ports = append(service.Object.Spec.Ports, servicePort)
				currentServicePortIndexes[port.HostPort] = len(service.Object.Spec.Ports) - 1
			}
		}
	}
	return nil
}
func (service *Service) GetValues() map[string]construct.IaCValue {
	return service.Values
}
