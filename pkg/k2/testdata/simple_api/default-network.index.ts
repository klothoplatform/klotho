import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'


const kloConfig = new pulumi.Config('klo')
const protect = kloConfig.getBoolean('protect') ?? false
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')
const accountId = pulumi.output(aws.getCallerIdentity({}))
const region = pulumi.output(aws.getRegion({}))

const default_network_private_subnet_1_route_table_nat_gateway_elastic_ip = new aws.ec2.Eip("default-network-private-subnet-1-route_table-nat_gateway-elastic_ip", {
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-1-route_table-nat_gateway-elastic_ip"},
    })
const default_network_private_subnet_2_route_table_nat_gateway_elastic_ip = new aws.ec2.Eip("default-network-private-subnet-2-route_table-nat_gateway-elastic_ip", {
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-2-route_table-nat_gateway-elastic_ip"},
    })
const region_0 = pulumi.output(aws.getRegion({}))
const default_network_vpc = new aws.ec2.Vpc("default-network-vpc", {
        cidrBlock: "10.0.0.0/16",
        enableDnsHostnames: true,
        enableDnsSupport: true,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-vpc"},
    })
const availability_zone_0 = pulumi.output(
        aws.getAvailabilityZones({
            state: 'available',
        })
    ).names[0]
const availability_zone_1 = pulumi.output(
        aws.getAvailabilityZones({
            state: 'available',
        })
    ).names[1]
const internet_gateway_0 = new aws.ec2.InternetGateway("internet_gateway-0", {
        vpcId: default_network_vpc.id,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "internet_gateway-0"},
    })
const default_network_private_subnet_1 = new aws.ec2.Subnet("default-network-private-subnet-1", {
        vpcId: default_network_vpc.id,
        cidrBlock: "10.0.128.0/18",
        availabilityZone: availability_zone_0,
        mapPublicIpOnLaunch: false,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-1"},
    })
const default_network_public_subnet_1 = new aws.ec2.Subnet("default-network-public-subnet-1", {
        vpcId: default_network_vpc.id,
        cidrBlock: "10.0.0.0/18",
        availabilityZone: availability_zone_0,
        mapPublicIpOnLaunch: false,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-public-subnet-1"},
    })
const default_network_private_subnet_2 = new aws.ec2.Subnet("default-network-private-subnet-2", {
        vpcId: default_network_vpc.id,
        cidrBlock: "10.0.192.0/18",
        availabilityZone: availability_zone_1,
        mapPublicIpOnLaunch: false,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-2"},
    })
const default_network_public_subnet_2 = new aws.ec2.Subnet("default-network-public-subnet-2", {
        vpcId: default_network_vpc.id,
        cidrBlock: "10.0.64.0/18",
        availabilityZone: availability_zone_1,
        mapPublicIpOnLaunch: false,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-public-subnet-2"},
    })
const default_network_public_subnet_1_route_table = new aws.ec2.RouteTable("default-network-public-subnet-1-route_table", {
        vpcId: default_network_vpc.id,
        routes: [
    {
        cidrBlock: "0.0.0.0/0",
        gatewayId: internet_gateway_0.id
    },
]

,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-public-subnet-1-route_table"},
    })
const default_network_public_subnet_2_route_table = new aws.ec2.RouteTable("default-network-public-subnet-2-route_table", {
        vpcId: default_network_vpc.id,
        routes: [
    {
        cidrBlock: "0.0.0.0/0",
        gatewayId: internet_gateway_0.id
    },
]

,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-public-subnet-2-route_table"},
    })
const default_network_private_subnet_1_route_table_nat_gateway = new aws.ec2.NatGateway("default-network-private-subnet-1-route_table-nat_gateway", {
        allocationId: default_network_private_subnet_1_route_table_nat_gateway_elastic_ip.id,
        subnetId: default_network_public_subnet_1.id,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-1-route_table-nat_gateway"},
    })
const default_network_private_subnet_2_route_table_nat_gateway = new aws.ec2.NatGateway("default-network-private-subnet-2-route_table-nat_gateway", {
        allocationId: default_network_private_subnet_2_route_table_nat_gateway_elastic_ip.id,
        subnetId: default_network_public_subnet_2.id,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-2-route_table-nat_gateway"},
    })
const default_network_public_subnet_1_default_network_public_subnet_1_route_table = new aws.ec2.RouteTableAssociation("default-network-public-subnet-1-default-network-public-subnet-1-route_table", {
        subnetId: default_network_public_subnet_1.id,
        routeTableId: default_network_public_subnet_1_route_table.id,
    })
const default_network_public_subnet_2_default_network_public_subnet_2_route_table = new aws.ec2.RouteTableAssociation("default-network-public-subnet-2-default-network-public-subnet-2-route_table", {
        subnetId: default_network_public_subnet_2.id,
        routeTableId: default_network_public_subnet_2_route_table.id,
    })
const default_network_private_subnet_1_route_table = new aws.ec2.RouteTable("default-network-private-subnet-1-route_table", {
        vpcId: default_network_vpc.id,
        routes: [
  {
    cidrBlock: "0.0.0.0/0",
    natGatewayId: default_network_private_subnet_1_route_table_nat_gateway.id
  },
]

,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-1-route_table"},
    })
const default_network_private_subnet_2_route_table = new aws.ec2.RouteTable("default-network-private-subnet-2-route_table", {
        vpcId: default_network_vpc.id,
        routes: [
  {
    cidrBlock: "0.0.0.0/0",
    natGatewayId: default_network_private_subnet_2_route_table_nat_gateway.id
  },
]

,
        tags: {GLOBAL_KLOTHO_TAG: "k2", RESOURCE_NAME: "default-network-private-subnet-2-route_table"},
    })
const default_network_private_subnet_1_default_network_private_subnet_1_route_table = new aws.ec2.RouteTableAssociation("default-network-private-subnet-1-default-network-private-subnet-1-route_table", {
        subnetId: default_network_private_subnet_1.id,
        routeTableId: default_network_private_subnet_1_route_table.id,
    })
const default_network_private_subnet_2_default_network_private_subnet_2_route_table = new aws.ec2.RouteTableAssociation("default-network-private-subnet-2-default-network-private-subnet-2-route_table", {
        subnetId: default_network_private_subnet_2.id,
        routeTableId: default_network_private_subnet_2_route_table.id,
    })

export const $outputs = {
}

export const $urns = {
	"aws:elastic_ip:default-network-private-subnet-1-route_table-nat_gateway-elastic_ip": (default_network_private_subnet_1_route_table_nat_gateway_elastic_ip as any).urn,
	"aws:elastic_ip:default-network-private-subnet-2-route_table-nat_gateway-elastic_ip": (default_network_private_subnet_2_route_table_nat_gateway_elastic_ip as any).urn,
	"aws:region:region-0": (region_0 as any).urn,
	"aws:vpc:default-network-vpc": (default_network_vpc as any).urn,
	"aws:availability_zone:region-0:availability_zone-0": (availability_zone_0 as any).urn,
	"aws:availability_zone:region-0:availability_zone-1": (availability_zone_1 as any).urn,
	"aws:internet_gateway:default-network-vpc:internet_gateway-0": (internet_gateway_0 as any).urn,
	"aws:subnet:default-network-vpc:default-network-private-subnet-1": (default_network_private_subnet_1 as any).urn,
	"aws:subnet:default-network-vpc:default-network-public-subnet-1": (default_network_public_subnet_1 as any).urn,
	"aws:subnet:default-network-vpc:default-network-private-subnet-2": (default_network_private_subnet_2 as any).urn,
	"aws:subnet:default-network-vpc:default-network-public-subnet-2": (default_network_public_subnet_2 as any).urn,
	"aws:route_table:default-network-vpc:default-network-public-subnet-1-route_table": (default_network_public_subnet_1_route_table as any).urn,
	"aws:route_table:default-network-vpc:default-network-public-subnet-2-route_table": (default_network_public_subnet_2_route_table as any).urn,
	"aws:nat_gateway:default-network-public-subnet-1:default-network-private-subnet-1-route_table-nat_gateway": (default_network_private_subnet_1_route_table_nat_gateway as any).urn,
	"aws:nat_gateway:default-network-public-subnet-2:default-network-private-subnet-2-route_table-nat_gateway": (default_network_private_subnet_2_route_table_nat_gateway as any).urn,
	"aws:route_table_association:default-network-public-subnet-1-default-network-public-subnet-1-route_table": (default_network_public_subnet_1_default_network_public_subnet_1_route_table as any).urn,
	"aws:route_table_association:default-network-public-subnet-2-default-network-public-subnet-2-route_table": (default_network_public_subnet_2_default_network_public_subnet_2_route_table as any).urn,
	"aws:route_table:default-network-vpc:default-network-private-subnet-1-route_table": (default_network_private_subnet_1_route_table as any).urn,
	"aws:route_table:default-network-vpc:default-network-private-subnet-2-route_table": (default_network_private_subnet_2_route_table as any).urn,
	"aws:route_table_association:default-network-private-subnet-1-default-network-private-subnet-1-route_table": (default_network_private_subnet_1_default_network_private_subnet_1_route_table as any).urn,
	"aws:route_table_association:default-network-private-subnet-2-default-network-private-subnet-2-route_table": (default_network_private_subnet_2_default_network_private_subnet_2_route_table as any).urn,
}
