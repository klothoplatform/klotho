package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	autoscaling "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	HorizontalPodAutoscaler struct {
		Name            string
		ConstructRefs   construct.BaseConstructSet `yaml:"-"`
		Object          *autoscaling.HorizontalPodAutoscaler
		Transformations map[string]construct.IaCValue
		FilePath        string
		Cluster         construct.ResourceId
	}
)

const (
	HORIZONTAL_POD_AUTOSCALER_TYPE = "horizontal_pod_autoscaler"
)

func (hpa *HorizontalPodAutoscaler) BaseConstructRefs() construct.BaseConstructSet {
	return hpa.ConstructRefs
}

func (hpa *HorizontalPodAutoscaler) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     HORIZONTAL_POD_AUTOSCALER_TYPE,
		Name:     hpa.Name,
	}
}

func (hpa *HorizontalPodAutoscaler) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (hpa *HorizontalPodAutoscaler) GetObject() v1.Object {
	return hpa.Object
}
func (hpa *HorizontalPodAutoscaler) Kind() string {
	return hpa.Object.Kind
}

func (hpa *HorizontalPodAutoscaler) Path() string {
	return hpa.FilePath
}

func (hpa *HorizontalPodAutoscaler) GetResourcesUsingHPA(dag *construct.ResourceGraph) []construct.Resource {
	var resources []construct.Resource
	for _, res := range dag.GetAllUpstreamResources(hpa) {
		if manifest, ok := res.(ManifestFile); ok {
			resources = append(resources, manifest)
		}
	}
	return resources
}

func (hpa *HorizontalPodAutoscaler) MakeOperational(dag *construct.ResourceGraph, appName string, classifier classification.Classifier) error {
	if hpa.Cluster.Name == "" {
		return fmt.Errorf("horizontal hpa autoscaler %s has no cluster", hpa.Name)
	}

	SetDefaultObjectMeta(hpa, hpa.Object.GetObjectMeta())
	hpa.FilePath = ManifestFilePath(hpa)
	return nil
}

func (hpa *HorizontalPodAutoscaler) GetValues() map[string]construct.IaCValue {
	return hpa.Transformations
}
