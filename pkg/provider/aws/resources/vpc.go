package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var elasticIpSanitizer = aws.SubnetSanitizer
var igwSanitizer = aws.SubnetSanitizer
var natGatewaySanitizer = aws.SubnetSanitizer
var vpcEndpointSanitizer = aws.SubnetSanitizer
var subnetSanitizer = aws.SubnetSanitizer

const (
	PrivateSubnet  = "private"
	PublicSubnet   = "public"
	IsolatedSubnet = "isolated"

	ELASTIC_IP_TYPE       = "elastic_ip"
	INTERNET_GATEWAY_TYPE = "internet_gateway"
	NAT_GATEWAY_TYPE      = "nat_gateway"
	VPC_SUBNET_TYPE       = "vpc_subnet"
	VPC_ENDPOINT_TYPE     = "vpc_endpoint"
	VPC_TYPE              = "vpc"

	CIDR_BLOCK_IAC_VALUE = "cidr_block"
)

type (
	Vpc struct {
		Name               string
		ConstructsRef      []core.AnnotationKey
		CidrBlock          string
		EnableDnsSupport   bool
		EnableDnsHostnames bool
	}
	ElasticIp struct {
		Name          string
		ConstructsRef []core.AnnotationKey
	}
	InternetGateway struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Vpc           *Vpc
	}
	NatGateway struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ElasticIp     *ElasticIp
		Subnet        *Subnet
	}
	Subnet struct {
		Name             string
		ConstructsRef    []core.AnnotationKey
		CidrBlock        string
		Vpc              *Vpc
		Type             string
		AvailabilityZone core.IaCValue
	}
	VpcEndpoint struct {
		Name            string
		ConstructsRef   []core.AnnotationKey
		Vpc             *Vpc
		Region          *Region
		ServiceName     string
		VpcEndpointType string
		Subnets         []*Subnet
	}
)

func CreateNetwork(config *config.Application, dag *core.ResourceGraph) *Vpc {
	appName := config.AppName
	vpc := NewVpc(appName)

	if dag.GetResource(vpc.Id()) != nil {
		return vpc
	}

	region := NewRegion()
	azs := NewAvailabilityZones()
	igw := NewInternetGateway(appName, "igw1", vpc)

	dag.AddResource(region)
	dag.AddDependency(azs, region)
	dag.AddResource(vpc)
	dag.AddDependency(vpc, region)
	dag.AddResource(igw)
	dag.AddDependency(igw, vpc)

	az1 := core.IaCValue{
		Resource: azs,
		Property: "0",
	}
	az2 := core.IaCValue{
		Resource: azs,
		Property: "1",
	}
	CreatePrivateSubnet(appName, "private1", az1, vpc, "10.0.0.0/18", dag)
	CreatePrivateSubnet(appName, "private2", az2, vpc, "10.0.64.0/18", dag)
	CreatePublicSubnet("public1", az1, vpc, "10.0.128.0/18", dag)
	CreatePublicSubnet("public2", az2, vpc, "10.0.192.0/18", dag)

	// VPC Endpoints are dependent upon the subnets so we need to ensure the subnets are created first
	CreateGatewayVpcEndpoint("s3", vpc, region, dag)
	CreateGatewayVpcEndpoint("dynamodb", vpc, region, dag)

	CreateInterfaceVpcEndpoint("lambda", vpc, region, dag)
	CreateInterfaceVpcEndpoint("sqs", vpc, region, dag)
	CreateInterfaceVpcEndpoint("sns", vpc, region, dag)
	CreateInterfaceVpcEndpoint("secretsmanager", vpc, region, dag)

	return vpc
}

func GetVpc(cfg *config.Application, dag *core.ResourceGraph) *Vpc {
	for _, r := range dag.ListResources() {
		if vpc, ok := r.(*Vpc); ok {
			return vpc
		}
	}
	return CreateNetwork(cfg, dag)
}

func VpcExists(dag *core.ResourceGraph) bool {
	for _, r := range dag.ListResources() {
		if _, ok := r.(*Vpc); ok {
			return true
		}
	}
	return false
}

func GetSubnets(cfg *config.Application, dag *core.ResourceGraph) (sns []*Subnet) {
	vpc := GetVpc(cfg, dag)
	return vpc.GetVpcSubnets(dag)
}

func (vpc *Vpc) GetSecurityGroups(dag *core.ResourceGraph) []*SecurityGroup {
	securityGroups := []*SecurityGroup{}
	downstreamDeps := dag.GetUpstreamResources(vpc)
	for _, dep := range downstreamDeps {
		if securityGroup, ok := dep.(*SecurityGroup); ok {
			securityGroups = append(securityGroups, securityGroup)
		}
	}
	return securityGroups
}

func CreatePrivateSubnet(appName string, subnetName string, az core.IaCValue, vpc *Vpc, cidrBlock string, dag *core.ResourceGraph) *Subnet {

	subnet := NewSubnet(subnetName, vpc, cidrBlock, PrivateSubnet, az)

	dag.AddResource(subnet)
	dag.AddDependency(subnet, vpc)
	dag.AddDependency(subnet, az.Resource)

	ip := NewElasticIp(appName, subnetName)

	dag.AddResource(ip)

	natGateway := NewNatGateway(appName, subnetName, subnet, ip)

	dag.AddResource(natGateway)
	dag.AddDependency(natGateway, subnet)
	dag.AddDependency(natGateway, ip)

	return subnet
}

func CreatePublicSubnet(subnetName string, az core.IaCValue, vpc *Vpc, cidrBlock string, dag *core.ResourceGraph) *Subnet {
	subnet := NewSubnet(subnetName, vpc, cidrBlock, PublicSubnet, az)
	dag.AddResource(subnet)
	dag.AddDependency(subnet, vpc)
	dag.AddDependency(subnet, az.Resource)
	return subnet
}

func CreateGatewayVpcEndpoint(service string, vpc *Vpc, region *Region, dag *core.ResourceGraph) {
	vpce := NewVpcEndpoint(service, vpc, "Gateway", region, nil)
	dag.AddResource(vpce)
	dag.AddDependency(vpce, vpc)
	dag.AddDependency(vpce, region)
}

func CreateInterfaceVpcEndpoint(service string, vpc *Vpc, region *Region, dag *core.ResourceGraph) {
	vpc_subnets := vpc.GetVpcSubnets(dag)
	subnets := []*Subnet{}
	for _, s := range vpc_subnets {
		if s.Type == PrivateSubnet {
			subnets = append(subnets, s)
		}
	}
	vpce := NewVpcEndpoint(service, vpc, "Interface", region, subnets)
	dag.AddResource(vpce)
	dag.AddDependency(vpce, vpc)
	dag.AddDependency(vpce, region)
	for _, subnet := range subnets {
		dag.AddDependency(vpce, subnet)
	}
}

func (vpc *Vpc) GetVpcSubnets(dag *core.ResourceGraph) []*Subnet {
	subnets := []*Subnet{}
	downstreamDeps := dag.GetUpstreamResources(vpc)
	for _, dep := range downstreamDeps {
		if subnet, ok := dep.(*Subnet); ok {
			subnets = append(subnets, subnet)
		}
	}
	return subnets
}

func (vpc *Vpc) GetPrivateSubnets(dag *core.ResourceGraph) []*Subnet {
	subnets := []*Subnet{}
	downstreamDeps := dag.GetUpstreamResources(vpc)
	for _, dep := range downstreamDeps {
		if subnet, ok := dep.(*Subnet); ok {
			if subnet.Type == PrivateSubnet {
				subnets = append(subnets, subnet)
			}
		}
	}
	return subnets
}

func NewElasticIp(appName string, ipName string) *ElasticIp {
	return &ElasticIp{
		Name: elasticIpSanitizer.Apply(fmt.Sprintf("%s-%s", appName, ipName)),
	}
}

// Provider returns name of the provider the resource is correlated to
func (subnet *ElasticIp) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *ElasticIp) KlothoConstructRef() []core.AnnotationKey {
	return subnet.ConstructsRef
}

// ID returns the id of the cloud resource
func (subnet *ElasticIp) Id() string {
	return fmt.Sprintf("%s:%s:%s", subnet.Provider(), ELASTIC_IP_TYPE, subnet.Name)
}

func NewInternetGateway(appName string, igwName string, vpc *Vpc) *InternetGateway {
	return &InternetGateway{
		Name: igwSanitizer.Apply(fmt.Sprintf("%s-%s", appName, igwName)),
		Vpc:  vpc,
	}
}

// Provider returns name of the provider the resource is correlated to
func (igw *InternetGateway) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (igw *InternetGateway) KlothoConstructRef() []core.AnnotationKey {
	return igw.ConstructsRef
}

// ID returns the id of the cloud resource
func (igw *InternetGateway) Id() string {
	return fmt.Sprintf("%s:%s:%s", igw.Provider(), INTERNET_GATEWAY_TYPE, igw.Name)
}

func NewNatGateway(appName string, natGatewayName string, subnet *Subnet, ip *ElasticIp) *NatGateway {
	return &NatGateway{
		Name:      natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", appName, natGatewayName)),
		ElasticIp: ip,
		Subnet:    subnet,
	}
}

// Provider returns name of the provider the resource is correlated to
func (natGateway *NatGateway) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (natGateway *NatGateway) KlothoConstructRef() []core.AnnotationKey {
	return natGateway.ConstructsRef
}

// ID returns the id of the cloud resource
func (natGateway *NatGateway) Id() string {
	return fmt.Sprintf("%s:%s:%s", natGateway.Provider(), NAT_GATEWAY_TYPE, natGateway.Name)
}

func NewSubnet(subnetName string, vpc *Vpc, cidrBlock string, subnetType string, availabilityZone core.IaCValue) *Subnet {
	return &Subnet{
		Name:             subnetSanitizer.Apply(fmt.Sprintf("%s-%s", vpc.Name, subnetName)),
		CidrBlock:        cidrBlock,
		Vpc:              vpc,
		Type:             subnetType,
		AvailabilityZone: availabilityZone,
	}
}

// Provider returns name of the provider the resource is correlated to
func (subnet *Subnet) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *Subnet) KlothoConstructRef() []core.AnnotationKey {
	return subnet.ConstructsRef
}

// ID returns the id of the cloud resource
func (subnet *Subnet) Id() string {
	return fmt.Sprintf("%s:%s:%s", subnet.Provider(), VPC_SUBNET_TYPE, subnet.Name)
}

func NewVpcEndpoint(service string, vpc *Vpc, endpointType string, region *Region, subnets []*Subnet) *VpcEndpoint {
	return &VpcEndpoint{
		Name:            vpcEndpointSanitizer.Apply(fmt.Sprintf("%s-%s", vpc.Name, service)),
		Vpc:             vpc,
		ServiceName:     service,
		VpcEndpointType: endpointType,
		Region:          region,
		Subnets:         subnets,
	}
}

// Provider returns name of the provider the resource is correlated to
func (vpce *VpcEndpoint) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpce *VpcEndpoint) KlothoConstructRef() []core.AnnotationKey {
	return vpce.ConstructsRef
}

// ID returns the id of the cloud resource
func (vpce *VpcEndpoint) Id() string {
	return fmt.Sprintf("%s:%s:%s", vpce.Provider(), VPC_ENDPOINT_TYPE, vpce.Name)
}

func NewVpc(appName string) *Vpc {
	return &Vpc{
		Name:               aws.VpcSanitizer.Apply(appName),
		CidrBlock:          "10.0.0.0/16",
		EnableDnsSupport:   true,
		EnableDnsHostnames: true,
	}
}

// Provider returns name of the provider the resource is correlated to
func (vpc *Vpc) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpc *Vpc) KlothoConstructRef() []core.AnnotationKey {
	return vpc.ConstructsRef
}

// ID returns the id of the cloud resource
func (vpc *Vpc) Id() string {
	return fmt.Sprintf("%s:%s:%s", vpc.Provider(), VPC_TYPE, vpc.Name)
}
