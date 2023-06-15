package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	autoscaling "k8s.io/api/autoscaling/v2"
)

type (
	HorizontalPodAutoscaler struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          autoscaling.HorizontalPodAutoscaler
		Transformations map[string]core.IaCValue
		FilePath        string
	}
)

const (
	HORIZONTAL_POD_AUTOSCALER_TYPE = "horizontal_pod_autoscaler"
)

func (deployment *HorizontalPodAutoscaler) BaseConstructsRef() core.BaseConstructSet {
	return deployment.ConstructRefs
}

func (deployment *HorizontalPodAutoscaler) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     HORIZONTAL_POD_AUTOSCALER_TYPE,
		Name:     deployment.Name,
	}
}

func (deployment *HorizontalPodAutoscaler) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}
