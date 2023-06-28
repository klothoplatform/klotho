package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	autoscaling "k8s.io/api/autoscaling/v2"
)

type (
	HorizontalPodAutoscaler struct {
		ConstructRefs   core.BaseConstructSet
		Object          autoscaling.HorizontalPodAutoscaler
		Transformations map[string]core.IaCValue
		FilePath        string
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
		Name:     hpa.Object.Name,
	}
}

func (hpa *HorizontalPodAutoscaler) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}
func (hpa *HorizontalPodAutoscaler) Kind() string {
	return hpa.Object.Kind
}

func (hpa *HorizontalPodAutoscaler) Path() string {
	return hpa.FilePath
}
