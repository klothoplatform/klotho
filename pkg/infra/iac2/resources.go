package iac2

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type (
	KubernetesProvider struct {
		ConstructsRef         core.BaseConstructSet
		KubeConfig            core.Resource
		Name                  string
		EnableServerSideApply bool
	}

	RouteTableAssociation struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Subnet        *resources.Subnet
		RouteTable    *resources.RouteTable
	}

	SecurityGroupRule struct {
		ConstructsRef   core.BaseConstructSet
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
		ConstructsRef  core.BaseConstructSet
		Name           string
		TargetGroupArn core.IaCValue
		TargetId       core.IaCValue
		Port           int
	}
)

func (e *KubernetesProvider) BaseConstructsRef() core.BaseConstructSet {
	return e.ConstructsRef
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

func (e *RouteTableAssociation) BaseConstructsRef() core.BaseConstructSet {
	return e.ConstructsRef
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
func (e *SecurityGroupRule) BaseConstructsRef() core.BaseConstructSet {
	return e.ConstructsRef
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
func (e *TargetGroupAttachment) BaseConstructsRef() core.BaseConstructSet {
	return e.ConstructsRef
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
