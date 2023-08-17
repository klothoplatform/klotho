package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	elbv2api "sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
)

type (
	TargetGroupBinding struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Object        *elbv2api.TargetGroupBinding
		Values        map[string]core.IaCValue
		FilePath      string
		Cluster       core.ResourceId
	}
)

const (
	TARGET_GROUP_BINDING_TYPE = "target_group_binding"
)

func (tgb *TargetGroupBinding) BaseConstructRefs() core.BaseConstructSet {
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

func (tgb *TargetGroupBinding) GetObject() v1.Object {
	return tgb.Object
}

func (tgb *TargetGroupBinding) Kind() string {
	return tgb.Object.Kind
}

func (tgb *TargetGroupBinding) Path() string {
	return tgb.FilePath
}

func (tgb *TargetGroupBinding) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if tgb.Cluster.Name == "" {
		return fmt.Errorf("target group binding %s has no cluster", tgb.Name)
	}
	if tgb.Object == nil {
		tgb.Object = &elbv2api.TargetGroupBinding{}
	}
	SetDefaultObjectMeta(tgb, tgb.Object.GetObjectMeta())
	tgb.FilePath = ManifestFilePath(tgb)
	return nil
}

func (tgb *TargetGroupBinding) GetValues() map[string]core.IaCValue {
	return tgb.Values
}
