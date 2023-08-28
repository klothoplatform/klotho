package iac2

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type (
	KubernetesProvider struct {
		ConstructRefs         construct.BaseConstructSet
		KubeConfig            construct.Resource
		Name                  string
		EnableServerSideApply bool
	}

	RouteTableAssociation struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Subnet        *resources.Subnet
		RouteTable    *resources.RouteTable
	}

	SecurityGroupRule struct {
		ConstructRefs   construct.BaseConstructSet
		Name            string
		Description     string
		FromPort        int
		ToPort          int
		Protocol        string
		CidrBlocks      []construct.IaCValue
		SecurityGroupId construct.IaCValue
		Type            string
	}

	TargetGroupAttachment struct {
		ConstructRefs  construct.BaseConstructSet
		Name           string
		TargetGroupArn construct.IaCValue
		TargetId       construct.IaCValue
		Port           int
	}
)

func (e *KubernetesProvider) BaseConstructRefs() construct.BaseConstructSet {
	return e.ConstructRefs
}

func (e *KubernetesProvider) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "pulumi",
		Type:     "kubernetes_provider",
		Name:     e.Name,
	}
}

func (f *KubernetesProvider) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{}
}

func (e *RouteTableAssociation) BaseConstructRefs() construct.BaseConstructSet {
	return e.ConstructRefs
}

func (e *RouteTableAssociation) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "pulumi",
		Type:     "route_table_association",
		Name:     e.Name,
	}
}
func (f *RouteTableAssociation) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{}
}
func (e *SecurityGroupRule) BaseConstructRefs() construct.BaseConstructSet {
	return e.ConstructRefs
}

func (e *SecurityGroupRule) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "pulumi",
		Type:     "security_group_rule",
		Name:     e.Name,
	}
}
func (f *SecurityGroupRule) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{}
}
func (e *TargetGroupAttachment) BaseConstructRefs() construct.BaseConstructSet {
	return e.ConstructRefs
}

func (e *TargetGroupAttachment) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: "pulumi",
		Type:     "target_group_attachment",
		Name:     e.Name,
	}
}
func (f *TargetGroupAttachment) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{}
}
