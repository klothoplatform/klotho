package resources

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
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
	zap.S().Debugf("Creating vpc %s", params.AppName)
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
	zap.S().Debugf("Creating internet gateway %s", params.AppName)
	igw.Name = igwSanitizer.Apply(fmt.Sprintf("%s-igw", params.AppName))
	igw.ConstructsRef = params.Refs.Clone()
	existingIgw := dag.GetResource(igw.Id())
	if existingIgw != nil {
		graphIgw := existingIgw.(*InternetGateway)
		graphIgw.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(igw)
	}
	return nil
}

func (igw *InternetGateway) MakeOperational(dag *core.ResourceGraph, appName string) error {
	zap.S().Debugf("Making internet gateway %s operational", igw.Name)
	if igw.Vpc == nil {
		vpcs := core.GetDownstreamResourcesOfType[*Vpc](dag, igw)
		if len(vpcs) > 1 {
			return fmt.Errorf("internet gateway %s has multiple vpc dependencies", igw.Name)
		} else if len(vpcs) == 0 {
			err := dag.CreateDependencies(igw, map[string]any{
				"Vpc": VpcCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(igw),
				},
			})
			if err != nil {
				return err
			}
		} else {
			igw.Vpc = vpcs[0]
		}
	}
	dag.AddDependenciesReflect(igw)
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
	nat.ConstructsRef = params.Refs.Clone()

	existingNat := dag.GetResource(nat.Id())
	if existingNat != nil {
		graphNat := existingNat.(*NatGateway)
		graphNat.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(nat)
	}
	return nil
}

func (nat *NatGateway) MakeOperational(dag *core.ResourceGraph, appName string) error {
	zap.S().Debugf("Making nat gateway %s operational", nat.Name)

	if nat.Subnet == nil {
		subnets := core.GetDownstreamResourcesOfType[*Subnet](dag, nat)
		if len(subnets) > 1 {
			return fmt.Errorf("nat gateway %s has multiple subnet dependencies", nat.Name)
		} else if len(subnets) == 0 {
			vpcs := core.GetDownstreamResourcesOfType[*Vpc](dag, nat)
			// Because private subnets depend on the nat we can use their dependency to see which vpc to place us in
			subnets := core.GetUpstreamResourcesOfType[*Subnet](dag, nat)
			for _, subnet := range subnets {
				if !collectionutil.Contains(vpcs, subnet.Vpc) {
					vpcs = append(vpcs, subnet.Vpc)
				}
			}
			if len(vpcs) > 1 {
				return fmt.Errorf("nat gateway %s has multiple vpc dependencies", nat.Name)
			} else if len(vpcs) == 0 {
				err := dag.CreateDependencies(nat, map[string]any{
					"Subnet": SubnetCreateParams{
						AppName: appName,
						Refs:    core.BaseConstructSetOf(nat),
						Type:    PublicSubnet,
					},
				})
				if err != nil {
					return err
				}
			} else {
				var az string
				var usedAzs []string
				currNats := core.GetResources[*NatGateway](dag)
				for _, currNat := range currNats {
					if currNat.Subnet != nil && currNat.Subnet.Vpc == vpcs[0] {
						usedAzs = append(usedAzs, currNat.Subnet.AvailabilityZone.PropertyVal)
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
				s, err := core.CreateResource[*Subnet](dag, SubnetCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(nat),
					AZ:      az,
					Type:    PublicSubnet,
				})
				if err != nil {
					return err
				}
				dag.AddDependency(s, vpcs[0])
				dag.AddDependency(nat, s)
				err = s.MakeOperational(dag, appName)
				if err != nil {
					return err
				}
				nat.Subnet = s
			}
		} else {
			nat.Subnet = subnets[0]
		}
	}

	if nat.ElasticIp == nil {
		var eip *ElasticIp
		for _, res := range dag.GetDownstreamResources(nat) {
			if upstreamEip, ok := res.(*ElasticIp); ok {
				if eip != nil {
					return fmt.Errorf("nat gateway %s has multiple elastic ip dependencies", nat.Name)
				}
				eip = upstreamEip
			}
		}
		if eip == nil {
			err := dag.CreateDependencies(nat, map[string]any{
				"ElasticIp": EipCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(nat),
					Name:    nat.Subnet.AvailabilityZone.PropertyVal,
				},
			})
			if err != nil {
				return err
			}
		} else {
			nat.ElasticIp = eip
		}
	}
	dag.AddDependenciesReflect(nat)
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
	subnet.ConstructsRef = params.Refs.Clone()
	subnet.Type = params.Type
	if params.AZ != "" {
		subnet.AvailabilityZone = &AwsResourceValue{ResourceVal: NewAvailabilityZones(), PropertyVal: params.AZ}
	}
	// We must check to see if there is an existent subnet after calling create dependencies because the id of the subnet has a namespace based on the vpc
	existingSubnet := dag.GetResource(subnet.Id())
	if existingSubnet != nil {
		graphSubnet := existingSubnet.(*Subnet)
		graphSubnet.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(subnet)
	}
	return nil
}

func (subnet *Subnet) MakeOperational(dag *core.ResourceGraph, appName string) error {
	copyOfSubnet := *subnet
	zap.S().Debugf("Making subnet %s operational", subnet.Name)
	var az string
	var usedAzs []string
	var typeToUse string
	var usedTypes []string
	subnetTypes := []string{PrivateSubnet, PublicSubnet}
	if subnet.Vpc == nil {
		var vpc *Vpc
		for _, res := range dag.GetDownstreamResources(subnet) {
			if upstreamVpc, ok := res.(*Vpc); ok {
				if vpc != nil {
					return fmt.Errorf("internet gateway %s has multiple vpc dependencies", subnet.Name)
				}
				vpc = upstreamVpc
			}
		}
		if vpc == nil {
			err := dag.CreateDependencies(subnet, map[string]any{
				"Vpc": VpcCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(subnet),
				},
			})
			if err != nil {
				return err
			}
		} else {
			subnet.Vpc = vpc
		}
		// Replace now that we are namespaced within the vpc
		dag.ReplaceConstruct(&copyOfSubnet, subnet)
		copyOfSubnet = *subnet

	}
	// Determine AZ and subnet type if they are not defined
	if subnet.AvailabilityZone == nil || subnet.Type == "" {
		currSubnets := core.GetResources[*Subnet](dag)

		if subnet.AvailabilityZone == nil {
			for _, currSubnet := range currSubnets {
				if currSubnet.AvailabilityZone != nil {
					usedAzs = append(usedAzs, currSubnet.AvailabilityZone.PropertyVal)
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
			subnet.AvailabilityZone = &AwsResourceValue{ResourceVal: NewAvailabilityZones(), PropertyVal: az}
		}
		if subnet.Type == "" {
			for _, currSubnet := range currSubnets {
				if currSubnet.Type != "" && currSubnet.AvailabilityZone != nil && currSubnet.AvailabilityZone.PropertyVal == subnet.AvailabilityZone.PropertyVal {
					usedTypes = append(usedTypes, currSubnet.Type)
				}
			}
			for _, subnetType := range subnetTypes {
				if !collectionutil.Contains(usedTypes, subnetType) {
					typeToUse = subnetType
				}
			}
			if typeToUse == "" {
				typeToUse = PrivateSubnet
			}
			subnet.Type = typeToUse
		}
		subnet.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s%s", appName, subnet.Type, subnet.AvailabilityZone.PropertyVal))
		// Replace now that we are namespaced within the vpc and determined the type and az of subnet
		dag.ReplaceConstruct(&copyOfSubnet, subnet)
	}

	routeTableFound := false
	for _, res := range dag.GetUpstreamResources(subnet) {
		if _, ok := res.(*RouteTable); ok {
			routeTableFound = true
			break
		}
	}
	if !routeTableFound {
		var rtName string
		if subnet.Type == PrivateSubnet {
			rtName = fmt.Sprintf("%s%s", subnet.Type, subnet.AvailabilityZone.PropertyVal)
		} else {
			rtName = fmt.Sprintf(subnet.Type)
		}
		routeTableParams := RouteTableCreateParams{
			AppName: appName,
			Refs:    core.BaseConstructSetOf(subnet),
			Name:    rtName,
		}

		rt, err := core.CreateResource[*RouteTable](dag, routeTableParams)
		if err != nil {
			return err
		}
		dag.AddDependency(rt, subnet)
		err = rt.MakeOperational(dag, appName)
		if err != nil {
			return err
		}
	}
	dag.AddDependenciesReflect(subnet)
	return nil
}

type SubnetConfigureParams struct {
}

func (subnet *Subnet) Configure(params SubnetConfigureParams) error {
	zap.S().Debugf("Configuring subnet %s", subnet.Name)
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
	zap.S().Debugf("Creating route table %s", params.Name)
	rt.Name = subnetSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	rt.ConstructsRef = params.Refs.Clone()
	// We must check to see if there is an existent route table after calling create dependencies because the id of the subnet can contain a namespace based on the vpc
	existingRt := dag.GetResource(rt.Id())
	if existingRt != nil {
		graphRt := existingRt.(*RouteTable)
		graphRt.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(rt)
	}
	return nil
}

func (routeTable *RouteTable) MakeOperational(dag *core.ResourceGraph, appName string) error {
	zap.S().Debugf("Making route table %s operational", routeTable.Name)

	routeTablesSubnets := core.GetDownstreamResourcesOfType[*Subnet](dag, routeTable)

	if routeTable.Vpc == nil {
		vpcs := core.GetAllDownstreamResourcesOfType[*Vpc](dag, routeTable)

		if len(vpcs) > 1 {
			return fmt.Errorf("route table %s has multiple vpc dependencies", routeTable.Name)
		}
		if len(vpcs) == 1 {
			routeTable.Vpc = vpcs[0]
		}
	}
	for _, subnet := range routeTablesSubnets {
		if routeTable.Vpc != nil && routeTable.Vpc != subnet.Vpc {
			return fmt.Errorf("route table %s has multiple vpc dependencies through its subnets", routeTable.Name)
		}
		routeTable.Vpc = subnet.Vpc
	}

	if routeTable.Vpc == nil {
		err := dag.CreateDependencies(routeTable, map[string]any{
			"Vpc": VpcCreateParams{
				AppName: appName,
				Refs:    core.BaseConstructSetOf(routeTable),
			},
		})
		if err != nil {
			return err
		}
	}
	for _, subnet := range routeTablesSubnets {
		if subnet.Type == PrivateSubnet {
			natAdded := false
			nats := core.GetUpstreamResourcesOfType[*NatGateway](dag, routeTable.Vpc)
			for _, nat := range nats {
				if nat.Subnet.AvailabilityZone.PropertyVal == subnet.AvailabilityZone.PropertyVal {
					natAdded = true
					dag.AddDependency(routeTable, nat)
				}
			}
			if !natAdded {
				natParams := NatCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(routeTable),
					Name:    subnet.AvailabilityZone.PropertyVal,
				}
				nat, err := core.CreateResource[*NatGateway](dag, natParams)
				if err != nil {
					return err
				}
				dag.AddDependency(routeTable, nat)
				for _, subnet := range routeTablesSubnets {
					dag.AddDependency(subnet, nat)
				}
				err = nat.MakeOperational(dag, appName)
				if err != nil {
					return err
				}
			}
		} else if subnet.Type == PublicSubnet {
			igwAdded := false
			igws := core.GetAllUpstreamResourcesOfType[*InternetGateway](dag, routeTable.Vpc)
			for _, igw := range igws {
				if igwAdded {
					return fmt.Errorf("route table %s has multiple internet gateway dependencies", routeTable.Name)
				}
				igwAdded = true
				dag.AddDependency(routeTable, igw)
			}
			if !igwAdded {
				igwParams := IgwCreateParams{
					AppName: appName,
					Refs:    core.BaseConstructSetOf(routeTable),
				}
				igw, err := core.CreateResource[*InternetGateway](dag, igwParams)
				if err != nil {
					return err
				}
				dag.AddDependency(routeTable, igw)
				dag.AddDependency(igw, routeTable.Vpc)
				err = igw.MakeOperational(dag, appName)
				if err != nil {
					return err
				}
			}
		}
	}
	dag.AddDependenciesReflect(routeTable)
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

func createSubnets(dag *core.ResourceGraph, appName string, ref core.Resource, vpc *Vpc) ([]*Subnet, error) {
	subnets := []*Subnet{}
	for i := 0; i < 4; i++ {

		azMarker := 0
		if i > 1 {
			azMarker = 1
		}
		typeMarker := PrivateSubnet
		if i%2 == 1 {
			typeMarker = PublicSubnet
		}
		subnet, err := core.CreateResource[*Subnet](dag, SubnetCreateParams{
			AppName: appName,
			Refs:    core.BaseConstructSetOf(ref),
			AZ:      availabilityZones[azMarker],
			Type:    typeMarker,
		})
		if err != nil {
			return subnets, err
		}
		if vpc != nil {
			dag.AddDependency(subnet, vpc)
		}
		err = subnet.MakeOperational(dag, appName)
		if err != nil {
			return subnets, err
		}
		subnets = append(subnets, subnet)
	}

	return subnets, nil
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
