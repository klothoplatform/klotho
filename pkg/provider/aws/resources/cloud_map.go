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
		ConstructsRef core.BaseConstructSet `yaml:"-"`
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
		dag.AddResource(namespace)
	}
	return nil
}

func (namespace *PrivateDnsNamespace) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if namespace.Vpc == nil {
		vpc, err := getSingleUpstreamVpc(dag, namespace)
		if err != nil {
			return err
		}
		if vpc == nil {
			err := dag.CreateDependencies(namespace, map[string]any{
				"Vpc": VpcCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(namespace),
				},
			})
			if err != nil {
				return err
			}
		} else {
			namespace.Vpc = vpc
			dag.AddDependency(namespace, vpc)
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

func (ns *PrivateDnsNamespace) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}
