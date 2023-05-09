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
	ROUTE_TABLE_TYPE      = "route_table"

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
		Name                string
		ConstructsRef       []core.AnnotationKey
		CidrBlock           string
		Vpc                 *Vpc
		Type                string
		AvailabilityZone    core.IaCValue
		MapPublicIpOnLaunch bool
	}
	VpcEndpoint struct {
		Name             string
		ConstructsRef    []core.AnnotationKey
		Vpc              *Vpc
		Region           *Region
		ServiceName      string
		VpcEndpointType  string
		Subnets          []*Subnet
		RouteTables      []*RouteTable
		SecurityGroupIds []core.IaCValue
	}
	RouteTable struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Vpc           *Vpc
		Routes        []*RouteTableRoute
	}
	RouteTableRoute struct {
		CidrBlock    string
		NatGatewayId core.IaCValue
		GatewayId    core.IaCValue
	}
)

func (vpc *Vpc) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type vpcMetadata struct {
		AppName string
		Refs    []core.AnnotationKey
	}
	data := &vpcMetadata{}
	decoder := getMapDecoder(data)
	err := decoder.Decode(metadata)
	if err != nil {
		return vpc, err
	}
	vpc = &Vpc{
		Name:               aws.VpcSanitizer.Apply(data.AppName),
		ConstructsRef:      data.Refs,
		CidrBlock:          "10.0.0.0/16",
		EnableDnsSupport:   true,
		EnableDnsHostnames: true,
	}

	existingVpc := core.GetResourceOfType[*Vpc](dag, vpc.Id().String())
	if existingVpc != nil {
		vpc = *existingVpc
		vpc.ConstructsRef = append(vpc.ConstructsRef, data.Refs...)
	} else {
		err = dag.CreateRecursively(vpc, metadata)
	}

	return vpc, err
}

func (eip *ElasticIp) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type eipMetadata struct {
		AppName string
		IpName  string
		Refs    []core.AnnotationKey
	}
	data := &eipMetadata{}
	decoder := getMapDecoder(data)
	err := decoder.Decode(metadata)
	if err != nil {
		return eip, err
	}

	eip = &ElasticIp{
		Name:          elasticIpSanitizer.Apply(fmt.Sprintf("%s-%s", data.AppName, data.IpName)),
		ConstructsRef: data.Refs,
	}
	existingEip := core.GetResourceOfType[*ElasticIp](dag, eip.Id().String())
	if existingEip != nil {
		eip = *existingEip
		eip.ConstructsRef = append(eip.ConstructsRef, data.Refs...)
	} else {
		err = dag.CreateRecursively(eip, metadata)
	}

	return eip, err
}

func (igw *InternetGateway) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type igwMetadata struct {
		AppName string
		Refs    []core.AnnotationKey
	}
	data := &igwMetadata{}
	decoder := getMapDecoder(data)
	err := decoder.Decode(metadata)
	if err != nil {
		return igw, err
	}

	igw = &InternetGateway{
		Name:          igwSanitizer.Apply(fmt.Sprintf("%s-igw", data.AppName)),
		ConstructsRef: data.Refs,
	}
	existingIgw := core.GetResourceOfType[*InternetGateway](dag, igw.Id().String())
	if existingIgw != nil {
		igw = *existingIgw
		igw.ConstructsRef = append(igw.ConstructsRef, data.Refs...)
	} else {
		err = dag.CreateRecursively(igw, metadata)
	}

	return igw, err
}

func (nat *NatGateway) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {

	type natMetadata struct {
		AppName string
		Refs    []core.AnnotationKey
		AZ      string
	}
	data := &natMetadata{}
	decoder := getMapDecoder(data)
	err := decoder.Decode(metadata)
	if err != nil {
		return nat, err
	}

	metadata["IpName"] = data.AZ

	nat = &NatGateway{
		Name:          natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", data.AppName, data.AZ)),
		ConstructsRef: data.Refs,
		Subnet: &Subnet{
			Type: PublicSubnet,
			AvailabilityZone: core.IaCValue{
				Resource: NewAvailabilityZones(),
				Property: data.AZ,
			},
		},
	}
	existingNat := core.GetResourceOfType[*NatGateway](dag, nat.Id().String())
	if existingNat != nil {
		nat = *existingNat
		nat.ConstructsRef = append(nat.ConstructsRef, data.Refs...)
	} else {
		err = dag.CreateRecursively(nat, metadata)
	}
	return nat, err
}

func (subnet *Subnet) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type subnetMetadata struct {
		AppName string
		Refs    []core.AnnotationKey
	}
	data := &subnetMetadata{}
	decoder := getMapDecoder(data)
	err := decoder.Decode(metadata)
	if err != nil {
		return subnet, err
	}

	if subnet.Type == "" {
		return subnet, fmt.Errorf("subnet type must be set for creation")
	}

	if subnet.AvailabilityZone.Property == "" {
		return subnet, fmt.Errorf("az must be set for creation")
	}

	subnet.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s%s", data.AppName, subnet.Type, subnet.AvailabilityZone.Property))
	subnet.ConstructsRef = data.Refs

	existingSubnet := core.GetResourceOfType[*Subnet](dag, subnet.Id().String())
	if existingSubnet != nil {
		subnet = *existingSubnet
		subnet.ConstructsRef = append(subnet.ConstructsRef, data.Refs...)
	} else {

		if subnet.Type == PrivateSubnet {
			nat := NatGateway{}
			metadata["AZ"] = subnet.AvailabilityZone.Property
			subnetNat, err := nat.Create(dag, metadata)
			metadata["Gateway"] = subnetNat
			if err != nil {
				return subnet, err
			}
			if subnet.AvailabilityZone.Property == "0" {
				subnet.CidrBlock = "10.0.0.0/18"
			} else if subnet.AvailabilityZone.Property == "1" {
				subnet.CidrBlock = "10.0.64.0/18"
			}
		} else if subnet.Type == PublicSubnet {
			igw := InternetGateway{}
			fullIgw, err := igw.Create(dag, metadata)
			metadata["Gateway"] = fullIgw

			if err != nil {
				return subnet, err
			}
			if subnet.AvailabilityZone.Property == "0" {
				subnet.CidrBlock = "10.0.128.0/18"
			} else if subnet.AvailabilityZone.Property == "1" {
				subnet.CidrBlock = "10.0.192.0/18"

			}
		}
		rt := RouteTable{}
		_, err := rt.Create(dag, metadata)
		if err != nil {
			return subnet, err
		}

		mapPublicIpOnLaunch := false
		if subnet.Type == PublicSubnet {
			mapPublicIpOnLaunch = true
		}

		subnet.MapPublicIpOnLaunch = mapPublicIpOnLaunch
		err = dag.CreateRecursively(subnet, metadata)
	}
	return subnet, err
}

func (lambda *VpcEndpoint) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func (rt *RouteTable) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	type routeTableMetadata struct {
		Gateway core.Resource
		Refs    []core.AnnotationKey
	}
	data := &routeTableMetadata{}
	decoder := getMapDecoder(data)
	err := decoder.Decode(metadata)
	if err != nil {
		return rt, err
	}

	rt = &RouteTable{
		Name:          data.Gateway.Id().Name,
		ConstructsRef: data.Refs,
		Routes: []*RouteTableRoute{
			{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{Resource: data.Gateway, Property: ID_IAC_VALUE}},
		},
	}
	existingRt := core.GetResourceOfType[*RouteTable](dag, rt.Id().String())
	if existingRt != nil {
		rt = *existingRt
		rt.ConstructsRef = append(rt.ConstructsRef, data.Refs...)
	} else {
		err = dag.CreateRecursively(rt, metadata)
	}
	return rt, err
}

// CreateNetwork takes in a config and uses the appName to create an aws network and inject it into the dag.
//
// The network consists of:
// - 1 Vpc
// - 1 Internet Gateway
// - 2 Public subnets, in different availability zones, which use the public route table.
// - 1 Public Route Table that includes a route to an internet gateway.
// - 2 Nat Gateways, each one sitting in its own public subnet.
// - 2 private subnets, with their own route table.
// - 2 Private Route Tables that include a route to one of the Nat Gateways.
func CreateNetwork(config *config.Application, dag *core.ResourceGraph) *Vpc {
	appName := config.AppName
	vpc := NewVpc(appName)

	if dag.GetResource(vpc.Id()) != nil {
		return vpc
	}

	region := NewRegion()
	azs := NewAvailabilityZones()
	igw := NewInternetGateway(appName, "igw1", vpc)
	publicRt := &RouteTable{
		Name: fmt.Sprintf("%s-public", vpc.Name),
		Vpc:  vpc,
		Routes: []*RouteTableRoute{
			{CidrBlock: "0.0.0.0/0", GatewayId: core.IaCValue{Resource: igw, Property: ID_IAC_VALUE}},
		},
	}
	dag.AddDependenciesReflect(publicRt)

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

	public1 := CreatePublicSubnet("public1", az1, vpc, "10.0.128.0/18", dag)

	ip1 := NewElasticIp(appName, "public1")
	dag.AddDependenciesReflect(ip1)
	natGateway1 := NewNatGateway(appName, "public1", public1, ip1)
	dag.AddDependenciesReflect(natGateway1)

	public2 := CreatePublicSubnet("public2", az2, vpc, "10.0.192.0/18", dag)

	ip2 := NewElasticIp(appName, "public2")
	dag.AddDependenciesReflect(ip2)
	natGateway2 := NewNatGateway(appName, "public2", public2, ip2)
	dag.AddDependenciesReflect(natGateway2)

	dag.AddDependency(publicRt, public1)
	dag.AddDependency(publicRt, public2)

	private1 := CreatePrivateSubnet(appName, "private1", az1, vpc, "10.0.0.0/18", dag)

	rt1 := &RouteTable{
		Name: private1.Name,
		Vpc:  vpc,
		Routes: []*RouteTableRoute{
			{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{Resource: natGateway1, Property: ID_IAC_VALUE}},
		},
	}
	dag.AddDependenciesReflect(rt1)
	dag.AddDependency(rt1, private1)

	private2 := CreatePrivateSubnet(appName, "private2", az2, vpc, "10.0.64.0/18", dag)
	rt2 := &RouteTable{
		Name: private2.Name,
		Vpc:  vpc,
		Routes: []*RouteTableRoute{
			{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{Resource: natGateway2, Property: ID_IAC_VALUE}},
		},
	}
	dag.AddDependenciesReflect(rt2)
	dag.AddDependency(rt2, private2)

	routeTables := []*RouteTable{publicRt, rt1, rt2}
	// VPC Endpoints are dependent upon the subnets so we need to ensure the subnets are created first
	CreateGatewayVpcEndpoint("s3", vpc, region, routeTables, dag)
	CreateGatewayVpcEndpoint("dynamodb", vpc, region, routeTables, dag)

	CreateInterfaceVpcEndpoint("lambda", vpc, region, dag, config)
	CreateInterfaceVpcEndpoint("sqs", vpc, region, dag, config)
	CreateInterfaceVpcEndpoint("sns", vpc, region, dag, config)
	CreateInterfaceVpcEndpoint("secretsmanager", vpc, region, dag, config)

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
	dag.AddDependenciesReflect(subnet)
	return subnet
}

func CreatePublicSubnet(subnetName string, az core.IaCValue, vpc *Vpc, cidrBlock string, dag *core.ResourceGraph) *Subnet {
	subnet := NewSubnet(subnetName, vpc, cidrBlock, PublicSubnet, az)
	dag.AddResource(subnet)
	dag.AddDependency(subnet, vpc)
	dag.AddDependency(subnet, az.Resource)
	return subnet
}

func CreateGatewayVpcEndpoint(service string, vpc *Vpc, region *Region, routeTables []*RouteTable, dag *core.ResourceGraph) {
	vpce := NewVpcEndpoint(service, vpc, "Gateway", region, nil)
	vpce.RouteTables = routeTables
	dag.AddDependenciesReflect(vpce)
}

func CreateInterfaceVpcEndpoint(service string, vpc *Vpc, region *Region, dag *core.ResourceGraph, config *config.Application) {
	vpcSubnets := vpc.GetVpcSubnets(dag)
	var subnets []*Subnet
	for _, s := range vpcSubnets {
		if s.Type == PrivateSubnet {
			subnets = append(subnets, s)
		}
	}
	vpce := NewVpcEndpoint(service, vpc, "Interface", region, subnets)
	sgs := vpc.GetSecurityGroups(dag)
	if len(sgs) == 0 {
		sgs = append(sgs, GetSecurityGroup(config, dag))
	}
	for _, sg := range sgs {
		vpce.SecurityGroupIds = append(vpce.SecurityGroupIds, core.IaCValue{
			Resource: sg,
			Property: ID_IAC_VALUE,
		})
	}
	dag.AddDependenciesReflect(vpce)
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *ElasticIp) KlothoConstructRef() []core.AnnotationKey {
	return subnet.ConstructsRef
}

// Id returns the id of the cloud resource
func (subnet *ElasticIp) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ELASTIC_IP_TYPE,
		Name:     subnet.Name,
	}
}

func NewInternetGateway(appName string, igwName string, vpc *Vpc) *InternetGateway {
	return &InternetGateway{
		Name: igwSanitizer.Apply(fmt.Sprintf("%s-%s", appName, igwName)),
		Vpc:  vpc,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (igw *InternetGateway) KlothoConstructRef() []core.AnnotationKey {
	return igw.ConstructsRef
}

// Id returns the id of the cloud resource
func (igw *InternetGateway) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     INTERNET_GATEWAY_TYPE,
		Name:     igw.Name,
	}
}

func NewNatGateway(appName string, natGatewayName string, subnet *Subnet, ip *ElasticIp) *NatGateway {
	return &NatGateway{
		Name:      natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", appName, natGatewayName)),
		ElasticIp: ip,
		Subnet:    subnet,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (natGateway *NatGateway) KlothoConstructRef() []core.AnnotationKey {
	return natGateway.ConstructsRef
}

// Id returns the id of the cloud resource
func (natGateway *NatGateway) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     NAT_GATEWAY_TYPE,
		Name:     natGateway.Name,
	}
}

func NewSubnet(subnetName string, vpc *Vpc, cidrBlock string, subnetType string, availabilityZone core.IaCValue) *Subnet {
	mapPublicIpOnLaunch := false
	if subnetType == PublicSubnet {
		mapPublicIpOnLaunch = true
	}
	return &Subnet{
		Name:                subnetSanitizer.Apply(fmt.Sprintf("%s-%s", vpc.Name, subnetName)),
		CidrBlock:           cidrBlock,
		Vpc:                 vpc,
		Type:                subnetType,
		AvailabilityZone:    availabilityZone,
		MapPublicIpOnLaunch: mapPublicIpOnLaunch,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *Subnet) KlothoConstructRef() []core.AnnotationKey {
	return subnet.ConstructsRef
}

// Id returns the id of the cloud resource
func (subnet *Subnet) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     VPC_SUBNET_TYPE,
		Name:     subnet.Name,
	}
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpce *VpcEndpoint) KlothoConstructRef() []core.AnnotationKey {
	return vpce.ConstructsRef
}

// Id returns the id of the cloud resource
func (vpce *VpcEndpoint) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     VPC_ENDPOINT_TYPE,
		Name:     vpce.Name,
	}
}

func NewVpc(appName string) *Vpc {
	return &Vpc{
		Name:               aws.VpcSanitizer.Apply(appName),
		CidrBlock:          "10.0.0.0/16",
		EnableDnsSupport:   true,
		EnableDnsHostnames: true,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpc *Vpc) KlothoConstructRef() []core.AnnotationKey {
	return vpc.ConstructsRef
}

// Id returns the id of the cloud resource
func (vpc *Vpc) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     VPC_TYPE,
		Name:     vpc.Name,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rt *RouteTable) KlothoConstructRef() []core.AnnotationKey {
	return rt.ConstructsRef
}

// Id returns the id of the cloud resource
func (rt *RouteTable) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ROUTE_TABLE_TYPE,
		Name:     rt.Name,
	}
}
