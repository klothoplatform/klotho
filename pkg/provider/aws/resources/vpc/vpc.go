package vpc

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const VPC_TYPE = "vpc"

var sanitizer = aws.VpcSanitizer

type (
	Vpc struct {
		Name               string
		ConstructsRef      []core.AnnotationKey
		CidrBlock          string
		enableDnsHostnames bool
		enableDnsSupport   bool
	}
)

func NewVpc(appName string) *Vpc {
	return &Vpc{
		Name:               sanitizer.Apply(appName),
		CidrBlock:          "10.0.0.0/16",
		enableDnsSupport:   true,
		enableDnsHostnames: true,
	}
}

// Provider returns name of the provider the resource is correlated to
func (vpc *Vpc) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpc *Vpc) KlothoConstructRef() []core.AnnotationKey {
	return vpc.ConstructsRef
}

// ID returns the id of the cloud resource
func (vpc *Vpc) Id() string {
	return fmt.Sprintf("%s:%s:%s", vpc.Provider(), VPC_TYPE, vpc.Name)
}

func CreateNetwork(appName string, dag *core.ResourceGraph) {
	vpc := NewVpc("test-app")
	region := resources.NewRegion()

	dag.AddResource(region)
	dag.AddResource(vpc)

	CreateGatewayVpcEndpoint("s3", vpc, region, dag)
	CreateGatewayVpcEndpoint("dynamodb", vpc, region, dag)

	CreateInterfaceVpcEndpoint("lambda", vpc, region, dag)
	CreateInterfaceVpcEndpoint("sqs", vpc, region, dag)
	CreateInterfaceVpcEndpoint("sns", vpc, region, dag)
	CreateInterfaceVpcEndpoint("secretsmanager", vpc, region, dag)

	CreatePrivateSubnet(appName, "private1", vpc, "10.0.0.0/18", dag)
	CreatePrivateSubnet(appName, "private2", vpc, "10.0.64.0/18", dag)
	CreatePublicSubnet("public1", vpc, "10.0.128.0/18", dag)
	CreatePublicSubnet("public2", vpc, "10.0.192.0/18", dag)

}
