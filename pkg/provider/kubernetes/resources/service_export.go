package resources

import (
	"fmt"
	cloudmap "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	ServiceExport struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Object        *cloudmap.ServiceExport
		FilePath      string
		Cluster       core.ResourceId
	}
)

const (
	SERVICE_EXPORT_TYPE = "service_export"
)

func (se *ServiceExport) BaseConstructRefs() core.BaseConstructSet {
	return se.ConstructRefs
}

func (se *ServiceExport) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     SERVICE_EXPORT_TYPE,
		Name:     se.Name,
	}
}

func (se *ServiceExport) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (se *ServiceExport) GetObject() v1.Object {
	return se.Object
}

func (se *ServiceExport) Kind() string {
	return "ServiceExport"
}

func (se *ServiceExport) Path() string {
	return se.FilePath
}

func (se *ServiceExport) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if se.Cluster.Name == "" {
		return fmt.Errorf("service export %s has no cluster", se.Name)
	}
	var downstreamService *Service
	for _, service := range core.GetDownstreamResourcesOfType[*Service](dag, se) {
		if downstreamService != nil {
			return fmt.Errorf("%s has multiple downstream services", se.Id())
		}
		downstreamService = service
	}

	if se.Object == nil {
		return fmt.Errorf("%s has no object", se.Id())
	}
	if downstreamService.Object == nil {
		return fmt.Errorf("%s has no object", downstreamService.Id())
	}
	SetDefaultObjectMeta(se, se.Object.GetObjectMeta())
	se.FilePath = ManifestFilePath(se)

	// Binds the service export to the service it's exporting
	se.Object.Name = downstreamService.Object.Name
	se.Object.Namespace = downstreamService.Object.Namespace

	return nil
}
