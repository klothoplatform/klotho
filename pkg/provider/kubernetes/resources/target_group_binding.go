package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	"k8s.io/apimachinery/pkg/runtime"
	elbv2api "sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
)

type (
	TargetGroupBinding struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *elbv2api.TargetGroupBinding
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.Resource
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
		Name:     tgb.Name,
	}
}

func (tgb *TargetGroupBinding) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstreamOrDownstream: true,
	}
}

func (tgb *TargetGroupBinding) GetObject() runtime.Object {
	return tgb.Object
}

func (tgb *TargetGroupBinding) Kind() string {
	return tgb.Object.Kind
}

func (tgb *TargetGroupBinding) Path() string {
	return tgb.FilePath
}

func (tgb *TargetGroupBinding) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if tgb.Object.Spec.TargetGroupARN == "" {
		return fmt.Errorf("target group binding %s has no target group arn", tgb.Id())
	}
	if tgb.Cluster == nil {
		var downstreamClustersFound []core.Resource
		for _, res := range dag.GetAllDownstreamResources(tgb) {
			if core.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound = append(downstreamClustersFound, res)
			}
		}
		if len(downstreamClustersFound) == 1 {
			tgb.Cluster = downstreamClustersFound[0]
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("target group binding %s has more than one cluster downstream", tgb.Id())
		}
	}
	return core.NewOperationalResourceError(tgb, []string{string(core.Cluster)}, fmt.Errorf("target group binding %s has no cluster's to use", tgb.Id()))
}
