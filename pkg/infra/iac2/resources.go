package iac2

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type (
	KubernetesProvider struct {
		ConstructsRef         []core.AnnotationKey
		KubeConfig            core.Resource
		Name                  string
		EnableServerSideApply bool
	}

	RouteTableAssociation struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Subnet        *resources.Subnet
		RouteTable    *resources.RouteTable
	}

	SecurityGroupRule struct {
		ConstructsRef   []core.AnnotationKey
		Name            string
		Description     string
		FromPort        int
		ToPort          int
		Protocol        string
		CidrBlocks      []string
		SecurityGroupId core.IaCValue
		Type            string
	}
)

func (e *KubernetesProvider) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *KubernetesProvider) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "kubernetes_provider",
		Name:     e.Name,
	}
}

func (e *RouteTableAssociation) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *RouteTableAssociation) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "route_table_association",
		Name:     e.Name,
	}
}

const (
	IAM_ROLE_POLICY_ATTACH_TYPE = "role_policy_attach"
)

type (
	RolePolicyAttachment struct {
		Name   string
		Policy *resources.IamPolicy
		Role   *resources.IamRole
	}
)

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *RolePolicyAttachment) KlothoConstructRef() []core.AnnotationKey {
	return nil
}

// Id returns the id of the cloud resource
func (role *RolePolicyAttachment) Id() core.ResourceId {
	return core.ResourceId{
		Provider: resources.AWS_PROVIDER,
		Type:     IAM_ROLE_POLICY_ATTACH_TYPE,
		Name:     role.Name,
	}
}

func (e *SecurityGroupRule) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *SecurityGroupRule) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "security_group_rule",
		Name:     e.Name,
	}
}
