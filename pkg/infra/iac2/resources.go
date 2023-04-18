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

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (role *RolePolicyAttachment) KlothoConstructRef() []core.AnnotationKey {
	return nil
}

// ID returns the id of the cloud resource
func (role *RolePolicyAttachment) Id() string {
	return fmt.Sprintf("%s:%s:%s", role.Provider(), IAM_ROLE_POLICY_ATTACH_TYPE, role.Name)
}
