package vpc

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const VPC_ENDPOINT_TYPE = "vpc_endpoint"

var vpcEndpointSanitizer = aws.SubnetSanitizer

type (
	VpcEndpoint struct {
		Name            string
		ConstructsRef   []core.AnnotationKey
		Vpc             *Vpc
		Region          *resources.Region
		ServiceName     string
		VpcEndpointType string
	}
)

func NewVpcEndpoint(service string, vpc *Vpc, endpointType string, region *resources.Region) *VpcEndpoint {
	return &VpcEndpoint{
		Name:            vpcEndpointSanitizer.Apply(fmt.Sprintf("%s-%s", vpc.Name, service)),
		Vpc:             vpc,
		ServiceName:     service,
		VpcEndpointType: endpointType,
		Region:          region,
	}
}

// Provider returns name of the provider the resource is correlated to
func (vpce *VpcEndpoint) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpce *VpcEndpoint) KlothoConstructRef() []core.AnnotationKey {
	return vpce.ConstructsRef
}

// ID returns the id of the cloud resource
func (vpce *VpcEndpoint) Id() string {
	return fmt.Sprintf("%s:%s:%s", vpce.Provider(), VPC_ENDPOINT_TYPE, vpce.Name)
}

func CreateGatewayVpcEndpoint(service string, vpc *Vpc, region *resources.Region, dag *core.ResourceGraph) {
	vpce := NewVpcEndpoint(service, vpc, "Gateway", region)
	dag.AddResource(vpce)
	dag.AddDependency(vpc, vpce)
	dag.AddDependency(region, vpce)
}

func CreateInterfaceVpcEndpoint(service string, vpc *Vpc, region *resources.Region, dag *core.ResourceGraph) {
	vpce := NewVpcEndpoint(service, vpc, "Interface", region)
	dag.AddResource(vpce)
	dag.AddDependency(vpc, vpce)
	dag.AddDependency(region, vpce)
}
