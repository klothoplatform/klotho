package resources

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"go.uber.org/zap"
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

	ELASTIC_IP_TYPE        = "elastic_ip"
	INTERNET_GATEWAY_TYPE  = "internet_gateway"
	NAT_GATEWAY_TYPE       = "nat_gateway"
	VPC_SUBNET_TYPE_PREFIX = "subnet_"
	VPC_ENDPOINT_TYPE      = "vpc_endpoint"
	VPC_TYPE               = "vpc"
	ROUTE_TABLE_TYPE       = "route_table"

	CIDR_BLOCK_IAC_VALUE = "cidr_block"
)

type (
	Vpc struct {
		Name               string
		ConstructsRef      core.AnnotationKeySet
		CidrBlock          string
		EnableDnsSupport   bool
		EnableDnsHostnames bool
	}
	ElasticIp struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
	}
	InternetGateway struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Vpc           *Vpc
	}
	NatGateway struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		ElasticIp     *ElasticIp
		Subnet        *Subnet
	}
	Subnet struct {
		Name                string
		ConstructsRef       core.AnnotationKeySet
		CidrBlock           string
		Vpc                 *Vpc
		Type                string
		AvailabilityZone    core.IaCValue
		MapPublicIpOnLaunch bool
	}
	VpcEndpoint struct {
		Name             string
		ConstructsRef    core.AnnotationKeySet
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
		ConstructsRef core.AnnotationKeySet
		Vpc           *Vpc
		Routes        []*RouteTableRoute
	}
	RouteTableRoute struct {
		CidrBlock    string
		NatGatewayId core.IaCValue
		GatewayId    core.IaCValue
	}
)

type VpcCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
}

func (vpc *Vpc) Create(dag *core.ResourceGraph, params VpcCreateParams) error {

	vpc.Name = aws.VpcSanitizer.Apply(params.AppName)
	vpc.ConstructsRef = params.Refs

	existingVpc := dag.GetResource(vpc.Id())
	if existingVpc != nil {
		graphVpc := existingVpc.(*Vpc)
		graphVpc.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(vpc)
	}

	return nil
}

type VpcConfigureParams struct {
}

func (vpc *Vpc) Configure(params VpcConfigureParams) error {
	vpc.CidrBlock = "10.0.0.0/16"
	vpc.EnableDnsSupport = true
	vpc.EnableDnsHostnames = true
	return nil
}

type EipCreateParams struct {
	AppName string
	IpName  string
	Refs    core.AnnotationKeySet
}

func (eip *ElasticIp) Create(dag *core.ResourceGraph, params EipCreateParams) error {
	eip.Name = elasticIpSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.IpName))
	eip.ConstructsRef = params.Refs
	existingEip := dag.GetResourceByVertexId(eip.Id().String())
	if existingEip != nil {
		graphEip := existingEip.(*ElasticIp)
		graphEip.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(eip)
	}
	return nil
}

type IgwCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
}

func (igw *InternetGateway) Create(dag *core.ResourceGraph, params IgwCreateParams) error {

	igw.Name = igwSanitizer.Apply(fmt.Sprintf("%s-igw", params.AppName))
	igw.ConstructsRef = params.Refs

	existingIgw := dag.GetResourceByVertexId(igw.Id().String())

	if existingIgw != nil {
		graphIgw := existingIgw.(*InternetGateway)
		graphIgw.ConstructsRef.AddAll(params.Refs)
	} else {
		err := dag.CreateDependencies(igw, map[string]any{
			"Vpc": params,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type NatCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	AZ      string
}

func (nat *NatGateway) Create(dag *core.ResourceGraph, params NatCreateParams) error {

	nat.Name = natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.AZ))
	nat.ConstructsRef = params.Refs

	existingNat := dag.GetResourceByVertexId(nat.Id().String())
	if existingNat != nil {
		graphNat := existingNat.(*NatGateway)
		graphNat.ConstructsRef.AddAll(params.Refs)
	} else {
		subResourceParams := map[string]any{
			"Subnet": SubnetCreateParams{
				AppName: params.AppName,
				Refs:    params.Refs,
				AZ:      params.AZ,
				Type:    PublicSubnet,
			},
			"ElasticIp": EipCreateParams{
				AppName: params.AppName,
				Refs:    params.Refs,
				IpName:  params.AZ,
			},
		}
		err := dag.CreateDependencies(nat, subResourceParams)
		return err
	}
	return nil
}

type SubnetCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	AZ      string
	Type    string
}

func (subnet *Subnet) Create(dag *core.ResourceGraph, params SubnetCreateParams) error {
	subnet.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s%s", params.AppName, params.Type, params.AZ))
	subnet.ConstructsRef = params.Refs
	subnet.AvailabilityZone = core.IaCValue{Resource: NewAvailabilityZones(), Property: params.AZ}
	subnet.Type = params.Type

	routeTableParams := RouteTableCreateParams{
		AppName: params.AppName,
		Refs:    params.Refs,
	}
	if subnet.Type == PrivateSubnet {
		routeTableParams.Name = fmt.Sprintf("%s%s", params.Type, params.AZ)
	} else {
		routeTableParams.Name = fmt.Sprintf(params.Type)
	}
	rt := &RouteTable{}
	err := rt.Create(dag, routeTableParams)
	if err != nil {
		return err
	}
	if subnet.Type == PrivateSubnet {
		nat := &NatGateway{}
		natParams := NatCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs,
			AZ:      params.AZ,
		}
		err := nat.Create(dag, natParams)
		if err != nil {
			return err
		}
		dag.AddDependency(rt, nat)
	} else if subnet.Type == PublicSubnet {
		igw := &InternetGateway{}
		igwParams := IgwCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs,
		}
		err := igw.Create(dag, igwParams)
		if err != nil {
			return err
		}
		dag.AddDependency(rt, igw)
	}

	err = dag.CreateDependencies(subnet, map[string]any{
		"Vpc": params,
	})
	if err != nil {
		return err
	}
	dag.AddDependency(subnet, NewAvailabilityZones())
	dag.AddDependency(rt, subnet)

	// We must check to see if there is an existent subnet after calling create dependencies because the id of the subnet has a namespace based on the vpc
	existingSubnet := dag.GetResourceByVertexId(subnet.Id().String())
	if existingSubnet != nil {
		graphSubnet := existingSubnet.(*Subnet)
		graphSubnet.ConstructsRef.AddAll(params.Refs)
	}
	return nil
}

type SubnetConfigureParams struct {
}

func (subnet *Subnet) Configure(params SubnetConfigureParams) error {
	if subnet.Type == PrivateSubnet {
		if subnet.AvailabilityZone.Property == "0" {
			subnet.CidrBlock = "10.0.0.0/18"
		} else if subnet.AvailabilityZone.Property == "1" {
			subnet.CidrBlock = "10.0.64.0/18"
		}
	} else if subnet.Type == PublicSubnet {
		if subnet.AvailabilityZone.Property == "0" {
			subnet.CidrBlock = "10.0.128.0/18"
		} else if subnet.AvailabilityZone.Property == "1" {
			subnet.CidrBlock = "10.0.192.0/18"

		}
		subnet.MapPublicIpOnLaunch = true
	}
	return nil
}

type RouteTableCreateParams struct {
	AppName string
	Name    string
	Refs    core.AnnotationKeySet
}

func (rt *RouteTable) Create(dag *core.ResourceGraph, params RouteTableCreateParams) error {

	rt.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))

	subParams := map[string]any{
		"Vpc": VpcCreateParams{
			AppName: params.AppName,
			Refs:    params.Refs,
		},
	}
	err := dag.CreateDependencies(rt, subParams)
	if err != nil {
		return err
	}
	dag.AddDependenciesReflect(rt)

	// We must check to see if there is an existent route table after calling create dependencies because the id of the subnet can contain a namespace based on the vpc
	existingRt := dag.GetResourceByVertexId(rt.Id().String())
	if existingRt != nil {
		graphRt := existingRt.(*RouteTable)
		graphRt.ConstructsRef.AddAll(params.Refs)
	}
	return nil
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
	dag.AddDependency(vpc, region)

	publicRt := &RouteTable{
		Name: fmt.Sprintf("%s-public", vpc.Name),
		Vpc:  vpc,
	}

	importId := config.Imports[vpc.Id()]
	if importId == "" {
		igw := NewInternetGateway(appName, "igw1", vpc)
		dag.AddDependenciesReflect(igw)

		publicRt.Routes = append(publicRt.Routes, &RouteTableRoute{CidrBlock: "0.0.0.0/0", GatewayId: core.IaCValue{Resource: igw, Property: ID_IAC_VALUE}})
	}
	dag.AddDependenciesReflect(publicRt)

	var importSubnetIds []core.ResourceId
	for id := range config.Imports {
		if id.Provider == AWS_PROVIDER && strings.HasPrefix(id.Type, VPC_SUBNET_TYPE_PREFIX) {
			importSubnetIds = append(importSubnetIds, id)
		}
	}

	if len(importSubnetIds) > 0 {
		for _, id := range importSubnetIds {
			sn := &Subnet{
				Name: id.Name,
				Vpc:  vpc,
			}
			err := sn.SetTypeFromId(id)
			if err != nil {
				//? Should we return an error here instead of warning?
				zap.S().Warnf("Failed to import subnet, ignoring: %v", err)
			} else {
				dag.AddDependenciesReflect(sn)
			}
		}
		// TODO either add or check the auxillary resources such as
		// gateways, route tables, etc.
	} else {
		azs := NewAvailabilityZones()
		dag.AddDependency(azs, region)
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
	}

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
func (subnet *ElasticIp) KlothoConstructRef() core.AnnotationKeySet {
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
func (igw *InternetGateway) KlothoConstructRef() core.AnnotationKeySet {
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
func (natGateway *NatGateway) KlothoConstructRef() core.AnnotationKeySet {
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
func (subnet *Subnet) KlothoConstructRef() core.AnnotationKeySet {
	return subnet.ConstructsRef
}

// Id returns the id of the cloud resource
func (subnet *Subnet) Id() core.ResourceId {
	id := core.ResourceId{
		Provider:  AWS_PROVIDER,
		Type:      VPC_SUBNET_TYPE_PREFIX + subnet.Type,
		Namespace: subnet.Vpc.Name,
		Name:      subnet.Name,
	}
	return id
}

func (subnet *Subnet) SetTypeFromId(id core.ResourceId) error {
	if id.Provider != AWS_PROVIDER || !strings.HasPrefix(id.Type, VPC_SUBNET_TYPE_PREFIX) {
		return fmt.Errorf("invalid id '%s' for partial subnet '%s'", id, subnet.Name)
	}
	if subnet.Vpc.Name != id.Namespace {
		return fmt.Errorf("invalid id '%s' not matching subnet vpc: %s in partial subnet '%s'", id, subnet.Vpc.Name, subnet.Name)
	}
	subnet.Type = strings.TrimPrefix(id.Type, VPC_SUBNET_TYPE_PREFIX)
	return nil
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
func (vpce *VpcEndpoint) KlothoConstructRef() core.AnnotationKeySet {
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
func (vpc *Vpc) KlothoConstructRef() core.AnnotationKeySet {
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
func (rt *RouteTable) KlothoConstructRef() core.AnnotationKeySet {
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
