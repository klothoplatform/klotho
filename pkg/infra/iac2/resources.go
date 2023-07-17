package iac2

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type (
	KubernetesProvider struct {
		ConstructRefs         core.BaseConstructSet
		KubeConfig            core.Resource
		Name                  string
		EnableServerSideApply bool
	}

	RouteTableAssociation struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Subnet        *resources.Subnet
		RouteTable    *resources.RouteTable
	}

	SecurityGroupRule struct {
		ConstructRefs   core.BaseConstructSet
		Name            string
		Description     string
		FromPort        int
		ToPort          int
		Protocol        string
		CidrBlocks      []core.IaCValue
		SecurityGroupId core.IaCValue
		Type            string
	}

	TargetGroupAttachment struct {
		ConstructRefs  core.BaseConstructSet
		Name           string
		TargetGroupArn core.IaCValue
		TargetId       core.IaCValue
		Port           int
	}
)

func (e *KubernetesProvider) BaseConstructRefs() core.BaseConstructSet {
	return e.ConstructRefs
}

func (e *KubernetesProvider) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "kubernetes_provider",
		Name:     e.Name,
	}
}

func (f *KubernetesProvider) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}

func (e *RouteTableAssociation) BaseConstructRefs() core.BaseConstructSet {
	return e.ConstructRefs
}

func (e *RouteTableAssociation) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "route_table_association",
		Name:     e.Name,
	}
}
func (f *RouteTableAssociation) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}
func (e *SecurityGroupRule) BaseConstructRefs() core.BaseConstructSet {
	return e.ConstructRefs
}

func (e *SecurityGroupRule) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "security_group_rule",
		Name:     e.Name,
	}
}
func (f *SecurityGroupRule) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}
func (e *TargetGroupAttachment) BaseConstructRefs() core.BaseConstructSet {
	return e.ConstructRefs
}

func (e *TargetGroupAttachment) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "target_group_attachment",
		Name:     e.Name,
	}
}
func (f *TargetGroupAttachment) DeleteContext() core.DeleteContext {
	return core.DeleteContext{}
}
