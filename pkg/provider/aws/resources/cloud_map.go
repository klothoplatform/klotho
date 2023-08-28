package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var privateDnsNamespaceSanitizer = aws.PrivateDnsNamespaceSanitizer

const (
	PRIVATE_DNS_NAMESPACE_TYPE = "private_dns_namespace"
)

type (
	PrivateDnsNamespace struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Vpc           *Vpc
	}
)

type PrivateDnsNamespaceCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
}

func (namespace *PrivateDnsNamespace) Create(dag *construct.ResourceGraph, params PrivateDnsNamespaceCreateParams) error {
	namespace.Name = privateDnsNamespaceSanitizer.Apply(fmt.Sprintf("%s_pdns", params.AppName))
	namespace.ConstructRefs = params.Refs.Clone()

	existingNamespace, found := construct.GetResource[*PrivateDnsNamespace](dag, namespace.Id())
	if found {
		existingNamespace.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(namespace)
	}
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ns *PrivateDnsNamespace) BaseConstructRefs() construct.BaseConstructSet {
	return ns.ConstructRefs
}

// Id returns the id of the cloud resource
func (ns *PrivateDnsNamespace) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     PRIVATE_DNS_NAMESPACE_TYPE,
		Name:     ns.Name,
	}
}

func (ns *PrivateDnsNamespace) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}
