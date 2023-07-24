package resources

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"go.uber.org/zap"
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
		ConstructRefs      core.BaseConstructSet `yaml:"-"`
		CidrBlock          string
		EnableDnsSupport   bool
		EnableDnsHostnames bool
	}
	ElasticIp struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
	}
	InternetGateway struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		Vpc           *Vpc
	}
	NatGateway struct {
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		ElasticIp     *ElasticIp
		Subnet        *Subnet
	}
	Subnet struct {
		Name                string
		ConstructRefs       core.BaseConstructSet `yaml:"-"`
		CidrBlock           string
		Vpc                 *Vpc
		Type                string
		AvailabilityZone    core.IaCValue
		MapPublicIpOnLaunch bool
	}
	VpcEndpoint struct {
		Name             string
		ConstructRefs    core.BaseConstructSet `yaml:"-"`
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
		ConstructRefs core.BaseConstructSet `yaml:"-"`
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
	Refs    core.BaseConstructSet
}

func (vpc *Vpc) Create(dag *core.ResourceGraph, params VpcCreateParams) error {
	zap.S().Debugf("Creating vpc %s", params.AppName)
	vpc.Name = aws.VpcSanitizer.Apply(params.AppName)
	vpc.ConstructRefs = params.Refs.Clone()

	existingVpc := dag.GetResource(vpc.Id())
	if existingVpc != nil {
		graphVpc := existingVpc.(*Vpc)
		graphVpc.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(vpc)
	}

	return nil
}

type VpcConfigureParams struct {
}

func (vpc *Vpc) Configure(params VpcConfigureParams) error {
	zap.S().Debugf("Configuring vpc %s", vpc.Name)
	vpc.CidrBlock = "10.0.0.0/16"
	vpc.EnableDnsSupport = true
	vpc.EnableDnsHostnames = true
	return nil
}

type EipCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (eip *ElasticIp) Create(dag *core.ResourceGraph, params EipCreateParams) error {
	zap.S().Debugf("Creating elastic ip %s", params.Name)
	eip.Name = elasticIpSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	eip.ConstructRefs = params.Refs.Clone()
	existingEip := dag.GetResource(eip.Id())
	if existingEip != nil {
		graphEip := existingEip.(*ElasticIp)
		graphEip.ConstructRefs.AddAll(params.Refs)
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
	zap.S().Debugf("Creating internet gateway %s", params.AppName)
	igw.Name = igwSanitizer.Apply(fmt.Sprintf("%s-igw", params.AppName))
	igw.ConstructRefs = params.Refs.Clone()
	existingIgw := dag.GetResource(igw.Id())
	if existingIgw != nil {
		graphIgw := existingIgw.(*InternetGateway)
		graphIgw.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(igw)
	}
	return nil
}

type NatCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (nat *NatGateway) Create(dag *core.ResourceGraph, params NatCreateParams) error {
	zap.S().Debugf("Creating nat gateway %s", params.Name)
	nat.Name = natGatewaySanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	nat.ConstructRefs = params.Refs.Clone()

	existingNat := dag.GetResource(nat.Id())
	if existingNat != nil {
		graphNat := existingNat.(*NatGateway)
		graphNat.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(nat)
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
	zap.S().Debugf("Creating subnet %s", params.AppName)
	subnet.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s%s", params.AppName, params.Type, params.AZ))
	subnet.ConstructRefs = params.Refs.Clone()
	subnet.Type = params.Type
	if params.AZ != "" {
		subnet.AvailabilityZone = core.IaCValue{ResourceId: NewAvailabilityZones().Id(), Property: params.AZ}
	}
	// We must check to see if there is an existent subnet after calling create dependencies because the id of the subnet has a namespace based on the vpc
	existingSubnet := dag.GetResource(subnet.Id())
	if existingSubnet != nil {
		graphSubnet := existingSubnet.(*Subnet)
		graphSubnet.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(subnet)
	}
	return nil
}

func (subnet *Subnet) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	copyOfSubnet := *subnet
	zap.S().Debugf("Making subnet %s operational", subnet.Name)
	var az string
	var usedAzs []string
	// Determine AZ and subnet type if they are not defined
	if subnet.AvailabilityZone.ResourceId.IsZero() || subnet.Type == "" {
		currSubnets := core.GetResources[*Subnet](dag)

		if subnet.AvailabilityZone.ResourceId.IsZero() {
			for _, currSubnet := range currSubnets {
				if currSubnet.AvailabilityZone.ResourceId.IsZero() {
					usedAzs = append(usedAzs, currSubnet.AvailabilityZone.Property)
				}
			}
			for _, availabilityZone := range availabilityZones {
				if !collectionutil.Contains(usedAzs, availabilityZone) {
					az = availabilityZone
				}
			}
			if az == "" {
				az = availabilityZones[0]
			}
			subnet.AvailabilityZone = core.IaCValue{ResourceId: NewAvailabilityZones().Id(), Property: az}
			dag.AddDependency(subnet, NewAvailabilityZones())
		}
		subnet.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s%s", appName, subnet.Type, subnet.AvailabilityZone.Property))
		// Replace now that we are namespaced within the vpc and determined the type and az of subnet
		err := dag.ReplaceConstruct(&copyOfSubnet, subnet)
		if err != nil {
			return err
		}
	}
	return nil
}

type SubnetConfigureParams struct {
}

func (subnet *Subnet) Configure(params SubnetConfigureParams) error {
	zap.S().Debugf("Configuring subnet %s", subnet.Name)
	switch subnet.Type {
	case PrivateSubnet:
		switch subnet.AvailabilityZone.Property {
		case "0":
			subnet.CidrBlock = "10.0.0.0/18"
		case "1":
			subnet.CidrBlock = "10.0.64.0/18"
		}
	case PublicSubnet:
		switch subnet.AvailabilityZone.Property {
		case "0":
			subnet.CidrBlock = "10.0.128.0/18"
		case "1":
			subnet.CidrBlock = "10.0.192.0/18"

		}
	}
	return nil
}

type RouteTableCreateParams struct {
	AppName string
	Name    string
	Refs    core.BaseConstructSet
}

func (rt *RouteTable) Create(dag *core.ResourceGraph, params RouteTableCreateParams) error {
	zap.S().Debugf("Creating route table %s", params.Name)
	rt.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	rt.ConstructRefs = params.Refs.Clone()
	// We must check to see if there is an existent route table after calling create dependencies because the id of the subnet can contain a namespace based on the vpc
	existingRt := dag.GetResource(rt.Id())
	if existingRt != nil {
		graphRt := existingRt.(*RouteTable)
		graphRt.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(rt)
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (eip *ElasticIp) BaseConstructRefs() core.BaseConstructSet {
	return eip.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (igw *InternetGateway) BaseConstructRefs() core.BaseConstructSet {
	return igw.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (natGateway *NatGateway) BaseConstructRefs() core.BaseConstructSet {
	return natGateway.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (subnet *Subnet) BaseConstructRefs() core.BaseConstructSet {
	return subnet.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpce *VpcEndpoint) BaseConstructRefs() core.BaseConstructSet {
	return vpce.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (vpc *Vpc) BaseConstructRefs() core.BaseConstructSet {
	return vpc.ConstructRefs
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

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (rt *RouteTable) BaseConstructRefs() core.BaseConstructSet {
	return rt.ConstructRefs
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
