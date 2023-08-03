package resources

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	autoscaling "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	HorizontalPodAutoscaler struct {
		Name            string
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		Object          *autoscaling.HorizontalPodAutoscaler
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.ResourceId
	}
)

const (
	HORIZONTAL_POD_AUTOSCALER_TYPE = "horizontal_pod_autoscaler"
)

func (hpa *HorizontalPodAutoscaler) BaseConstructRefs() core.BaseConstructSet {
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

func (hpa *HorizontalPodAutoscaler) GetResourcesUsingHPA(dag *core.ResourceGraph) []core.Resource {
	var resources []core.Resource
	for _, res := range dag.GetAllUpstreamResources(hpa) {
		if manifest, ok := res.(ManifestFile); ok {
			resources = append(resources, manifest)
		}
	}
	return resources
}

func (hpa *HorizontalPodAutoscaler) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if hpa.Cluster.Name == "" {
		return fmt.Errorf("horizontal hpa autoscaler %s has no cluster", hpa.Name)
	}

	SetDefaultObjectMeta(hpa, hpa.Object.GetObjectMeta())
	hpa.FilePath = ManifestFilePath(hpa, hpa.Cluster)
	return nil
}

func (hpa *HorizontalPodAutoscaler) Values() map[string]core.IaCValue {
	return hpa.Transformations
}
