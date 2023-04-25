package iac2

import (
	"fmt"

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

func (e *KubernetesProvider) Provider() string {
	return "pulumi"
}

func (e *KubernetesProvider) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *KubernetesProvider) Id() string {
	return fmt.Sprintf("%s:%s:%s", e.Provider(), "kubernetes_provider", e.Name)
}

func (e *RouteTableAssociation) Provider() string {
	return "pulumi"
}

func (e *RouteTableAssociation) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *RouteTableAssociation) Id() string {
	return fmt.Sprintf("%s:%s:%s", e.Provider(), "route_table_association", e.Name)
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

// Provider returns name of the provider the resource is correlated to
func (role *RolePolicyAttachment) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *RolePolicyAttachment) KlothoConstructRef() []core.AnnotationKey {
	return nil
}

// Id returns the id of the cloud resource
func (role *RolePolicyAttachment) Id() string {
	return fmt.Sprintf("%s:%s:%s", role.Provider(), IAM_ROLE_POLICY_ATTACH_TYPE, role.Name)
}

func (e *SecurityGroupRule) Provider() string {
	return "pulumi"
}

func (e *SecurityGroupRule) KlothoConstructRef() []core.AnnotationKey {
	return e.ConstructsRef
}

func (e *SecurityGroupRule) Id() string {
	return fmt.Sprintf("%s:%s:%s", e.Provider(), "security_group_rule", e.Name)
}
