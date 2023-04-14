package resources

import (
	"fmt"

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
		ConstructsRef []core.AnnotationKey
		Vpc           *Vpc
	}
)

func NewPrivateDnsNamespace(appName string, refs []core.AnnotationKey, vpc *Vpc) *PrivateDnsNamespace {
	return &PrivateDnsNamespace{
		Name:          privateDnsNamespaceSanitizer.Apply(appName),
		ConstructsRef: refs,
		Vpc:           vpc,
	}
}

// Provider returns name of the provider the resource is correlated to
func (ns *PrivateDnsNamespace) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ns *PrivateDnsNamespace) KlothoConstructRef() []core.AnnotationKey {
	return ns.ConstructsRef
}

// ID returns the id of the cloud resource
func (ns *PrivateDnsNamespace) Id() string {
	return fmt.Sprintf("%s:%s:%s", ns.Provider(), PRIVATE_DNS_NAMESPACE_TYPE, ns.Name)
}
