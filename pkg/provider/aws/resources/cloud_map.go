package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var privateDnsNamespaceSanitizer = aws.PrivateDnsNamespaceSanitizer

const (
	PRIVATE_DNS_NAMESPACE_TYPE = "private_dns_namespace"
)

type (
	PrivateDnsNamespace struct {
		Name          string
		ConstructsRef core.BaseConstructSet
		Vpc           *Vpc
	}
)

type PrivateDnsNamespaceCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
}

func (namespace *PrivateDnsNamespace) Create(dag *core.ResourceGraph, params PrivateDnsNamespaceCreateParams) error {
	namespace.Name = privateDnsNamespaceSanitizer.Apply(params.AppName)
	namespace.ConstructsRef = params.Refs.Clone()

	existingNamespace, found := core.GetResource[*PrivateDnsNamespace](dag, namespace.Id())
	if found {
		existingNamespace.ConstructsRef.AddAll(params.Refs)
	} else {
		err := dag.CreateDependencies(namespace, map[string]any{"Vpc": VpcCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs.Clone(),
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ns *PrivateDnsNamespace) BaseConstructsRef() core.BaseConstructSet {
	return ns.ConstructsRef
}

// Id returns the id of the cloud resource
func (ns *PrivateDnsNamespace) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     PRIVATE_DNS_NAMESPACE_TYPE,
		Name:     ns.Name,
	}
}
