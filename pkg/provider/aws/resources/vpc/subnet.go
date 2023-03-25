package vpc

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const VPC_SUBNET_TYPE = "vpc_subnet"

var subnetSanitizer = aws.SubnetSanitizer

const (
	PrivateSubnet  = "private"
	PublicSubnet   = "public"
	IsolatedSubnet = "isolated"
)

type (
	Subnet struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		CidrBlock     string
		Vpc           *Vpc
		Type          string
	}
)

func NewSubnet(subnetName string, vpc *Vpc, cidrBlock string, subnetType string) *Subnet {
	return &Subnet{
		Name:      subnetSanitizer.Apply(fmt.Sprintf("%s-%s", vpc.Name, subnetName)),
		CidrBlock: cidrBlock,
		Vpc:       vpc,
		Type:      subnetType,
	}
}

// Provider returns name of the provider the resource is correlated to
func (subnet *Subnet) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *Subnet) KlothoConstructRef() []core.AnnotationKey {
	return subnet.ConstructsRef
}

// ID returns the id of the cloud resource
func (subnet *Subnet) Id() string {
	return fmt.Sprintf("%s:%s:%s", subnet.Provider(), VPC_SUBNET_TYPE, subnet.Name)
}

func CreatePrivateSubnet(appName string, subnetName string, vpc *Vpc, cidrBlock string, dag *core.ResourceGraph) {

	subnet := NewSubnet(subnetName, vpc, cidrBlock, PrivateSubnet)

	dag.AddResource(subnet)
	dag.AddDependency(vpc, subnet)

	ip := NewElasticIp(appName, subnetName)

	dag.AddResource(ip)
	dag.AddDependency(subnet, ip)

	natGateway := NewNatGateway(appName, subnetName, subnet, ip)

	dag.AddResource(natGateway)
	dag.AddDependency(subnet, natGateway)
	dag.AddDependency(ip, natGateway)
}

func CreatePublicSubnet(subnetName string, vpc *Vpc, cidrBlock string, dag *core.ResourceGraph) {
	subnet := NewSubnet(subnetName, vpc, cidrBlock, PublicSubnet)
	dag.AddResource(subnet)
	dag.AddDependency(vpc, subnet)
}
