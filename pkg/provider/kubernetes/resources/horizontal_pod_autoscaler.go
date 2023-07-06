package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	autoscaling "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	HorizontalPodAutoscaler struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *autoscaling.HorizontalPodAutoscaler
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.Resource
	}
)

const (
	HORIZONTAL_POD_AUTOSCALER_TYPE = "horizontal_pod_autoscaler"
)

func (hpa *HorizontalPodAutoscaler) BaseConstructsRef() core.BaseConstructSet {
	return hpa.ConstructRefs
}

func (hpa *HorizontalPodAutoscaler) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     HORIZONTAL_POD_AUTOSCALER_TYPE,
		Name:     hpa.Name,
	}
}

func (hpa *HorizontalPodAutoscaler) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (hpa *HorizontalPodAutoscaler) GetObject() runtime.Object {
	return hpa.Object
}
func (hpa *HorizontalPodAutoscaler) Kind() string {
	return hpa.Object.Kind
}

func (hpa *HorizontalPodAutoscaler) Path() string {
	return hpa.FilePath
}

func (hpa *HorizontalPodAutoscaler) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if hpa.Cluster == nil {
		downstreamClustersFound := map[string]core.Resource{}
		for _, res := range dag.GetAllDownstreamResources(hpa) {
			if core.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound[res.Id().String()] = res
			}
		}
		// See which cluster any pods or deployments using this service account use
		for _, res := range hpa.GetResourcesUsingHPA(dag) {
			for _, dres := range dag.GetAllDownstreamResources(res) {
				if core.GetFunctionality(dres) == core.Cluster {
					downstreamClustersFound[dres.Id().String()] = dres
				}
			}
		}

		if len(downstreamClustersFound) == 1 {
			_, cluster := collectionutil.GetOneEntry(downstreamClustersFound)
			hpa.Cluster = cluster
			dag.AddDependency(hpa, cluster)
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("horizontal pod autoscaler %s has more than one cluster downstream", hpa.Id())
		}

		return core.NewOperationalResourceError(hpa, []string{string(core.Cluster)}, fmt.Errorf("horizontal pod autoscaler %s has no clusters to use", hpa.Id()))
	}
	return nil
}

func (hpa *HorizontalPodAutoscaler) GetResourcesUsingHPA(dag *core.ResourceGraph) []core.Resource {
	var resources []core.Resource
	for _, res := range dag.GetAllUpstreamResources(hpa) {
		if manifest, ok := res.(ManifestFile); ok {
			resources = append(resources, manifest)
		}
	}
	return resources
}
