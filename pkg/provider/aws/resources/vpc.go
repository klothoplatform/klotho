package resources

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var elasticIpSanitizer = aws.SubnetSanitizer
var igwSanitizer = aws.SubnetSanitizer
var natGatewaySanitizer = aws.SubnetSanitizer
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
		ConstructsRef      core.BaseConstructSet `yaml:"-"`
		CidrBlock          string
		EnableDnsSupport   bool
		EnableDnsHostnames bool
	}
	ElasticIp struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}
	InternetGateway struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Vpc           *Vpc
	}
	NatGateway struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		ElasticIp     *ElasticIp
		Subnet        *Subnet
	}
	Subnet struct {
		Name                string
		ConstructsRef       core.BaseConstructSet `yaml:"-"`
		CidrBlock           string
		Vpc                 *Vpc
		Type                string
		AvailabilityZone    *AwsResourceValue
		MapPublicIpOnLaunch bool
	}
	VpcEndpoint struct {
		Name             string
		ConstructsRef    core.BaseConstructSet `yaml:"-"`
		Vpc              *Vpc
		Region           *Region
		ServiceName      string
		VpcEndpointType  string
		Subnets          []*Subnet
		RouteTables      []*RouteTable
		SecurityGroupIds []*AwsResourceValue
	}
	RouteTable struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Vpc           *Vpc
		Routes        []*RouteTableRoute
	}
	RouteTableRoute struct {
		CidrBlock    string
		NatGatewayId *AwsResourceValue
		GatewayId    *AwsResourceValue
	}
)

type VpcCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
}

func (vpc *Vpc) Create(dag *core.ResourceGraph, params VpcCreateParams) error {

	vpc.Name = aws.VpcSanitizer.Apply(params.AppName)
	vpc.ConstructsRef = params.Refs.Clone()

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
	Refs    core.BaseConstructSet
}

func (eip *ElasticIp) Create(dag *core.ResourceGraph, params EipCreateParams) error {
	eip.Name = elasticIpSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.IpName))
	eip.ConstructsRef = params.Refs.Clone()
	existingEip := dag.GetResource(eip.Id())
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
	Refs    core.BaseConstructSet
}

func (igw *InternetGateway) Create(dag *core.ResourceGraph, params IgwCreateParams) error {

	igw.Name = igwSanitizer.Apply(fmt.Sprintf("%s-igw", params.AppName))
	igw.ConstructsRef = params.Refs.Clone()

	existingIgw := dag.GetResource(igw.Id())

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
	Refs    core.BaseConstructSet
	AZ      string
}

func (nat *NatGateway) Create(dag *core.ResourceGraph, params NatCreateParams) error {

	nat.Name = natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.AZ))
	nat.ConstructsRef = params.Refs.Clone()

	existingNat := dag.GetResource(nat.Id())
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
	Refs    core.BaseConstructSet
	AZ      string
	Type    string
}

func (subnet *Subnet) Create(dag *core.ResourceGraph, params SubnetCreateParams) error {
	subnet.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s%s", params.AppName, params.Type, params.AZ))
	subnet.ConstructsRef = params.Refs.Clone()
	subnet.AvailabilityZone = &AwsResourceValue{ResourceVal: NewAvailabilityZones(), PropertyVal: params.AZ}
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
	existingSubnet := dag.GetResource(subnet.Id())
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
		if subnet.AvailabilityZone.PropertyVal == "0" {
			subnet.CidrBlock = "10.0.0.0/18"
		} else if subnet.AvailabilityZone.PropertyVal == "1" {
			subnet.CidrBlock = "10.0.64.0/18"
		}
	} else if subnet.Type == PublicSubnet {
		if subnet.AvailabilityZone.PropertyVal == "0" {
			subnet.CidrBlock = "10.0.128.0/18"
		} else if subnet.AvailabilityZone.PropertyVal == "1" {
			subnet.CidrBlock = "10.0.192.0/18"

		}
		subnet.MapPublicIpOnLaunch = true
	}
	return nil
}

type RouteTableCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (rt *RouteTable) Create(dag *core.ResourceGraph, params RouteTableCreateParams) error {
	rt.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	rt.ConstructsRef = params.Refs.Clone()

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
	existingRt := dag.GetResource(rt.Id())
	if existingRt != nil {
		graphRt := existingRt.(*RouteTable)
		graphRt.ConstructsRef.AddAll(params.Refs)
	}
	return nil
}

func VpcExists(dag *core.ResourceGraph) bool {
	for _, r := range dag.ListResources() {
		if _, ok := r.(*Vpc); ok {
			return true
		}
	}
	return false
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

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (eip *ElasticIp) BaseConstructsRef() core.BaseConstructSet {
	return eip.ConstructsRef
}

// Id returns the id of the cloud resource
func (eip *ElasticIp) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ELASTIC_IP_TYPE,
		Name:     eip.Name,
	}
}

func (eip *ElasticIp) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (igw *InternetGateway) BaseConstructsRef() core.BaseConstructSet {
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

func (igw *InternetGateway) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (natGateway *NatGateway) BaseConstructsRef() core.BaseConstructSet {
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
func (natGateway *NatGateway) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *Subnet) BaseConstructsRef() core.BaseConstructSet {
	return subnet.ConstructsRef
}

// Id returns the id of the cloud resource
func (subnet *Subnet) Id() core.ResourceId {
	id := core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     VPC_SUBNET_TYPE_PREFIX + subnet.Type,
		Name:     subnet.Name,
	}
	if subnet.Vpc != nil {
		id.Namespace = subnet.Vpc.Name
	}
	return id
}

func (subnet *Subnet) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (subnet *Subnet) Load(namespace string, dag *core.ConstructGraph) error {
	namespacedVpc := &Vpc{Name: namespace}
	vpc := dag.GetConstruct(namespacedVpc.Id())
	if vpc == nil {
		return fmt.Errorf("cannot load subnet with name %s because namespace vpc %s does not exist", subnet.Name, namespace)
	}
	if vpc, ok := vpc.(*Vpc); !ok {
		return fmt.Errorf("cannot load subnet with name %s because namespace vpc %s is not a vpc", subnet.Name, namespace)
	} else {
		subnet.Vpc = vpc
	}
	return nil
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

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpce *VpcEndpoint) BaseConstructsRef() core.BaseConstructSet {
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

func (vpc *VpcEndpoint) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpc *Vpc) BaseConstructsRef() core.BaseConstructSet {
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

func (vpc *Vpc) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rt *RouteTable) BaseConstructsRef() core.BaseConstructSet {
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

func (rt *RouteTable) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
