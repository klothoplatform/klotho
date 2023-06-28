package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	elbv2api "sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
)

type (
	TargetGroupBinding struct {
		ConstructRefs   core.BaseConstructSet
		Object          elbv2api.TargetGroupBinding
		Transformations map[string]core.IaCValue
		FilePath        string
	}
)

const (
	TARGET_GROUP_BINDING_TYPE = "target_group_binding"
)

func (tgb *TargetGroupBinding) BaseConstructsRef() core.BaseConstructSet {
	return tgb.ConstructRefs
}

func (tgb *TargetGroupBinding) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     TARGET_GROUP_BINDING_TYPE,
		Name:     tgb.Object.Name,
	}
}

func (tgb *TargetGroupBinding) OutputYAML() core.File {
	var outputFile core.File
	return outputFile
}

func (tgb *TargetGroupBinding) Kind() string {
	return tgb.Object.Kind
}

func (tgb *TargetGroupBinding) Path() string {
	return tgb.FilePath
}
