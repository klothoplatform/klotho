package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	corev1 "k8s.io/api/core/v1"
)

type (
	Service struct {
		ConstructRefs   core.BaseConstructSet
		Object          corev1.Service
		Transformations map[string]core.IaCValue
		FilePath        string
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
		Name:     service.Object.Name,
	}
}
func (service *Service) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}

func (service *Service) Kind() string {
	return service.Object.Kind
}

func (service *Service) Path() string {
	return service.FilePath
}
