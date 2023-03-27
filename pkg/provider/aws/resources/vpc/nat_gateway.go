package vpc

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const NAT_GATEWAY_TYPE = "nat_gateway"

var natGatewaySanitizer = aws.SubnetSanitizer

type (
	NatGateway struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ElasticIp     *ElasticIp
		Subnet        *Subnet
	}
)

func NewNatGateway(appName string, natGatewayName string, subnet *Subnet, ip *ElasticIp) *NatGateway {
	return &NatGateway{
		Name:      natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", appName, natGatewayName)),
		ElasticIp: ip,
		Subnet:    subnet,
	}
}

// Provider returns name of the provider the resource is correlated to
func (natGateway *NatGateway) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (natGateway *NatGateway) KlothoConstructRef() []core.AnnotationKey {
	return natGateway.ConstructsRef
}

// ID returns the id of the cloud resource
func (natGateway *NatGateway) Id() string {
	return fmt.Sprintf("%s:%s:%s", natGateway.Provider(), NAT_GATEWAY_TYPE, natGateway.Name)
}
