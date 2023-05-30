package iac2

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type (
	KubernetesProvider struct {
		ConstructsRef         core.AnnotationKeySet
		KubeConfig            core.Resource
		Name                  string
		EnableServerSideApply bool
	}

	RouteTableAssociation struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Subnet        *resources.Subnet
		RouteTable    *resources.RouteTable
	}

	SecurityGroupRule struct {
		ConstructsRef   core.AnnotationKeySet
		Name            string
		Description     string
		FromPort        int
		ToPort          int
		Protocol        string
		CidrBlocks      []core.IaCValue
		SecurityGroupId core.IaCValue
		Type            string
	}
)

func (e *KubernetesProvider) KlothoConstructRef() core.AnnotationKeySet {
	return e.ConstructsRef
}

func (e *KubernetesProvider) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "kubernetes_provider",
		Name:     e.Name,
	}
}

func (e *RouteTableAssociation) KlothoConstructRef() core.AnnotationKeySet {
	return e.ConstructsRef
}

func (e *RouteTableAssociation) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "route_table_association",
		Name:     e.Name,
	}
}

func (e *SecurityGroupRule) KlothoConstructRef() core.AnnotationKeySet {
	return e.ConstructsRef
}

func (e *SecurityGroupRule) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "security_group_rule",
		Name:     e.Name,
	}
}
