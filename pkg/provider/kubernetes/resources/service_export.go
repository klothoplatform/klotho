package resources

import (
	"fmt"

	cloudmap "github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/apis/multicluster/v1alpha1"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	ServiceExport struct {
		Name          string
		ConstructRefs core.BaseConstructSet
		Object        *cloudmap.ServiceExport
		FilePath      string
		Cluster       core.Resource
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

func (sa *ServiceExport) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (se *ServiceExport) GetObject() runtime.Object {
	return se.Object
}

func (se *ServiceExport) Kind() string {
	return "ServiceExport"
}

func (se *ServiceExport) Path() string {
	return se.FilePath
}

func (se *ServiceExport) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if se.Cluster == nil {
		var downstreamClustersFound []core.Resource
		for _, res := range dag.GetAllDownstreamResources(se) {
			if classifier.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound = append(downstreamClustersFound, res)
			}
		}
		if len(downstreamClustersFound) == 1 {
			se.Cluster = downstreamClustersFound[0]
			dag.AddDependency(se, se.Cluster)
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("service export %s has more than one cluster downstream", se.Id())
		}

		return core.NewOperationalResourceError(se, []string{string(core.Cluster)}, fmt.Errorf("service export %s has no clusters to use", se.Id()))
	}
	return nil
}
