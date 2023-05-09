package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_VpcCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		vpc  *Vpc
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil vpc",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name: "existing vpc",
			vpc:  &Vpc{Name: "my_app", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.vpc != nil {
				dag.AddResource(tt.vpc)
			}
			metadata := map[string]any{
				"AppName": "my-app",
				"Refs":    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			}

			updatedVpc, err := tt.vpc.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			vpc, ok := updatedVpc.(*Vpc)
			if !assert.True(ok) {
				return
			}

			assert.Equal(vpc.Name, "my_app")

			if tt.vpc == nil {
				assert.Equal(vpc.CidrBlock, "10.0.0.0/16")
				assert.True(vpc.EnableDnsHostnames)
				assert.True(vpc.EnableDnsSupport)
				assert.Equal(vpc.ConstructsRef, metadata["Refs"])
			} else {
				assert.Equal(vpc.CidrBlock, tt.vpc.CidrBlock)
				assert.Equal(vpc.EnableDnsHostnames, tt.vpc.EnableDnsHostnames)
				assert.Equal(vpc.EnableDnsSupport, tt.vpc.EnableDnsSupport)
				assert.Equal(vpc.ConstructsRef, tt.vpc.ConstructsRef)
			}
		})
	}
}

func Test_ElasticIpCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		eip  *ElasticIp
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil eip",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:my_app_ip0",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name: "existing eip",
			eip:  &ElasticIp{Name: "my_app_ip0", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:my_app_ip0",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.eip != nil {
				dag.AddResource(tt.eip)
			}
			metadata := map[string]any{
				"AppName": "my-app",
				"IpName":  "ip0",
				"Refs":    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			}

			updatedEip, err := tt.eip.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			eip, ok := updatedEip.(*ElasticIp)
			if !assert.True(ok) {
				return
			}

			assert.Equal(eip.Name, "my_app_ip0")
			if tt.eip == nil {
				assert.Equal(eip.ConstructsRef, metadata["Refs"])
			} else {
				assert.Equal(eip.ConstructsRef, tt.eip.ConstructsRef)
			}
		})
	}
}

func Test_InternetGatewayCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		igw  *InternetGateway
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil eip",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing eip",
			igw:  &InternetGateway{Name: "my_app_igw", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.igw != nil {
				dag.AddResource(tt.igw)
			}
			metadata := map[string]any{
				"AppName": "my-app",
				"Refs":    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
			}

			updatedIgw, err := tt.igw.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			igw, ok := updatedIgw.(*InternetGateway)
			if !assert.True(ok) {
				return
			}

			assert.Equal(igw.Name, "my_app_igw")
			if tt.igw == nil {
				assert.NotNil(igw.Vpc)
				assert.Equal(igw.ConstructsRef, metadata["Refs"])
			} else {
				assert.Nil(igw.Vpc)
				assert.Equal(igw.ConstructsRef, tt.igw.ConstructsRef)
			}
		})
	}
}

func Test_NatGatewayCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		nat  *NatGateway
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil eip",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:my_app_igw",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:vpc_subnet:my_app_public0"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing eip",
			nat:  &NatGateway{Name: "my_app_0", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:nat_gateway:my_app_0",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.nat != nil {
				dag.AddResource(tt.nat)
			}
			metadata := map[string]any{
				"AppName": "my-app",
				"Refs":    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				"AZ":      "0",
			}

			updatedNat, err := tt.nat.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			nat, ok := updatedNat.(*NatGateway)
			if !assert.True(ok) {
				return
			}

			assert.Equal(nat.Name, "my_app_0")
			if tt.nat == nil {
				assert.NotNil(nat.Subnet)
				assert.NotNil(nat.ElasticIp)
				assert.Equal(nat.ConstructsRef, metadata["Refs"])
			} else {
				assert.Nil(nat.Subnet)
				assert.Nil(nat.ElasticIp)
				assert.Equal(nat.ConstructsRef, tt.nat.ConstructsRef)
			}
		})
	}
}

func Test_SubnetCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name     string
		subnet   *Subnet
		addToDag bool
		want     coretesting.ResourcesExpectation
		wantErr  bool
	}{
		{
			name:     "private subnet az0",
			subnet:   &Subnet{Type: PrivateSubnet, AvailabilityZone: core.IaCValue{Property: "0"}, CidrBlock: "10.0.0.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:my_app_0",
					"aws:route_table:my_app_igw",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_private0",
					"aws:vpc_subnet:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:vpc_subnet:my_app_public0"},
					{Source: "aws:route_table:my_app_0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name:     "private subnet az1",
			subnet:   &Subnet{Type: PrivateSubnet, AvailabilityZone: core.IaCValue{Property: "1"}, CidrBlock: "10.0.64.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_1",
					"aws:route_table:my_app_igw",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_private1",
					"aws:vpc_subnet:my_app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:vpc_subnet:my_app_public1"},
					{Source: "aws:route_table:my_app_1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name:     "public subnet az0",
			subnet:   &Subnet{Type: PublicSubnet, AvailabilityZone: core.IaCValue{Property: "0"}, CidrBlock: "10.0.128.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_app_igw",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name:     "public subnet az1",
			subnet:   &Subnet{Type: PublicSubnet, AvailabilityZone: core.IaCValue{Property: "1"}, CidrBlock: "10.0.192.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_app_igw",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name:     "existing subnet",
			subnet:   &Subnet{Name: "my_app_private0", Type: PrivateSubnet, AvailabilityZone: core.IaCValue{Property: "0"}, ConstructsRef: initialRefs},
			addToDag: true,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc_subnet:my_app_private0",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:    "no type set for subnet",
			subnet:  &Subnet{AvailabilityZone: core.IaCValue{Property: "0"}},
			wantErr: true,
		},
		{
			name:    "no az set for subnet",
			subnet:  &Subnet{Type: PrivateSubnet},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.addToDag {
				dag.AddResource(tt.subnet)
			}
			metadata := map[string]any{
				"AppName": "my-app",
				"Refs":    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				"AZ":      tt.subnet.AvailabilityZone.Property,
			}

			updatedSubnet, err := tt.subnet.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			subnet, ok := updatedSubnet.(*Subnet)

			if !assert.True(ok) {
				return
			}

			assert.Equal(subnet.Name, fmt.Sprintf("my_app_%s%s", tt.subnet.Type, tt.subnet.AvailabilityZone.Property))
			assert.Equal(subnet.CidrBlock, tt.subnet.CidrBlock)
			assert.Equal(subnet.Type, tt.subnet.Type)
			assert.Equal(subnet.AvailabilityZone.Property, tt.subnet.AvailabilityZone.Property)
			if tt.subnet.Type == PublicSubnet {
				assert.True(subnet.MapPublicIpOnLaunch)
			} else {
				assert.False(subnet.MapPublicIpOnLaunch)
			}
			if tt.addToDag == false {
				assert.Equal(subnet.ConstructsRef, metadata["Refs"])
			} else {
				assert.Nil(subnet.Vpc)
				assert.Equal(subnet.ConstructsRef, tt.subnet.ConstructsRef)
			}
		})
	}
}

func Test_RouteTableCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		rt   *RouteTable
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil route table",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:0",
					"aws:route_table:my_app_igw",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:vpc_subnet:my_app_public0"},
					{Source: "aws:route_table:0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing rt",
			rt:   &RouteTable{Name: "0", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:0",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.rt != nil {
				dag.AddResource(tt.rt)
			}
			gw := &NatGateway{Name: "0"}
			metadata := map[string]any{
				"AppName": "my-app",
				"AZ":      "0",
				"Refs":    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				"Gateway": gw,
			}

			updatedRouteTable, err := tt.rt.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			routeTable, ok := updatedRouteTable.(*RouteTable)

			if !assert.True(ok) {
				return
			}

			assert.Equal(routeTable.Name, "0")

			if tt.rt == nil {
				assert.NotNil(routeTable.Vpc)
				assert.Equal(routeTable.Routes[0].CidrBlock, "0.0.0.0/0")
				assert.Equal(routeTable.Routes[0].NatGatewayId.Resource.Id().String(), "aws:nat_gateway:my_app_0")
				assert.Equal(routeTable.ConstructsRef, metadata["Refs"])
			} else {
				assert.Nil(routeTable.Vpc)
				assert.Equal(routeTable.ConstructsRef, tt.rt.ConstructsRef)
			}
		})
	}
}

func Test_CreateNetwork(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name              string
		existingResources []core.Resource
		want              coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:test_app_public1",
					"aws:elastic_ip:test_app_public2",
					"aws:internet_gateway:test_app_igw1",
					"aws:nat_gateway:test_app_public1",
					"aws:nat_gateway:test_app_public2",
					"aws:region:region",
					"aws:route_table:test_app-public",
					"aws:route_table:test_app_private1",
					"aws:route_table:test_app_private2",
					"aws:vpc:test_app",
					"aws:security_group:test-app",
					"aws:vpc_endpoint:test_app_dynamodb",
					"aws:vpc_endpoint:test_app_lambda",
					"aws:vpc_endpoint:test_app_s3",
					"aws:vpc_endpoint:test_app_secretsmanager",
					"aws:vpc_endpoint:test_app_sns",
					"aws:vpc_endpoint:test_app_sqs",
					"aws:vpc_subnet:test_app_private1",
					"aws:vpc_subnet:test_app_private2",
					"aws:vpc_subnet:test_app_public1",
					"aws:vpc_subnet:test_app_public2",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:availability_zones:AvailabilityZones", Destination: "aws:region:region"},
					{Source: "aws:internet_gateway:test_app_igw1", Destination: "aws:vpc:test_app"},
					{Source: "aws:nat_gateway:test_app_public1", Destination: "aws:elastic_ip:test_app_public1"},
					{Source: "aws:nat_gateway:test_app_public1", Destination: "aws:vpc_subnet:test_app_public1"},
					{Source: "aws:nat_gateway:test_app_public2", Destination: "aws:elastic_ip:test_app_public2"},
					{Source: "aws:nat_gateway:test_app_public2", Destination: "aws:vpc_subnet:test_app_public2"},
					{Source: "aws:route_table:test_app-public", Destination: "aws:internet_gateway:test_app_igw1"},
					{Source: "aws:route_table:test_app-public", Destination: "aws:vpc:test_app"},
					{Source: "aws:route_table:test_app-public", Destination: "aws:vpc_subnet:test_app_public1"},
					{Source: "aws:route_table:test_app-public", Destination: "aws:vpc_subnet:test_app_public2"},
					{Source: "aws:route_table:test_app_private1", Destination: "aws:nat_gateway:test_app_public1"},
					{Source: "aws:route_table:test_app_private1", Destination: "aws:vpc:test_app"},
					{Source: "aws:route_table:test_app_private1", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:route_table:test_app_private2", Destination: "aws:nat_gateway:test_app_public2"},
					{Source: "aws:route_table:test_app_private2", Destination: "aws:vpc:test_app"},
					{Source: "aws:route_table:test_app_private2", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc:test_app", Destination: "aws:region:region"},
					{Source: "aws:security_group:test-app", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:route_table:test_app-public"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:route_table:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_dynamodb", Destination: "aws:route_table:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:security_group:test-app"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_lambda", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:route_table:test_app-public"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:route_table:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:route_table:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:security_group:test-app"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_secretsmanager", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:security_group:test-app"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_sns", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:region:region"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:security_group:test-app"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:vpc_subnet:test_app_private1"},
					{Source: "aws:vpc_endpoint:test_app_sqs", Destination: "aws:vpc_subnet:test_app_private2"},
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_private2", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_private2", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_public2", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_public2", Destination: "aws:vpc:test_app"},
				},
			},
		},
		{
			name:              "happy path",
			existingResources: []core.Resource{NewVpc(appName)},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app"},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			for _, res := range tt.existingResources {
				dag.AddResource(res)
			}

			cfg := &config.Application{
				AppName: appName,
			}
			result := CreateNetwork(cfg, dag)
			assert.NotNil(result)

			tt.want.Assert(t, dag)
		})
	}
}

func Test_GetVpcSubnets(t *testing.T) {

	type subnetSpec struct {
		Cidr   string
		Public bool
	}

	cases := []struct {
		name    string
		subnets []subnetSpec
	}{
		{
			name: "happy path",
			subnets: []subnetSpec{
				{"10.0.1.0/24", false},
				{"10.0.2.0/24", false},
				{"10.0.3.0/24", true},
				{"10.0.4.0/24", true},
			},
		},
		{
			name: "no subnets",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			vpc := NewVpc("test-app")
			dag.AddResource(vpc)
			for i, spec := range tt.subnets {
				if spec.Public {
					CreatePublicSubnet(fmt.Sprintf("public%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				} else {
					CreatePrivateSubnet("test-app", fmt.Sprintf("private%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				}
			}

			result := vpc.GetVpcSubnets(dag)
			var got []subnetSpec
			for _, sn := range result {
				got = append(got, subnetSpec{Cidr: sn.CidrBlock, Public: sn.Type == PublicSubnet})
			}
			assert.ElementsMatch(got, tt.subnets)
		})
	}
}

func Test_GetPrivateSubnets(t *testing.T) {

	type subnetSpec struct {
		Cidr   string
		Public bool
	}

	cases := []struct {
		name    string
		subnets []subnetSpec
		want    []subnetSpec
	}{
		{
			name: "happy path",
			subnets: []subnetSpec{
				{"10.0.1.0/24", false},
				{"10.0.2.0/24", false},
				{"10.0.3.0/24", true},
				{"10.0.4.0/24", true},
			},
			want: []subnetSpec{
				{"10.0.1.0/24", false}, {"10.0.2.0/24", false},
			},
		},
		{
			name: "no subnets",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			vpc := NewVpc("test-app")
			dag.AddResource(vpc)
			for i, spec := range tt.subnets {
				if spec.Public {
					CreatePublicSubnet(fmt.Sprintf("public%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				} else {
					CreatePrivateSubnet("test-app", fmt.Sprintf("private%d", i), core.IaCValue{Resource: NewAvailabilityZones()}, vpc, spec.Cidr, dag)
				}
			}

			result := vpc.GetPrivateSubnets(dag)
			var got []subnetSpec
			for _, sn := range result {
				got = append(got, subnetSpec{Cidr: sn.CidrBlock, Public: sn.Type == PublicSubnet})
			}
			assert.ElementsMatch(got, tt.want)
		})
	}
}

func Test_CreatePrivateSubnet(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:test_app",
					"aws:availability_zones:AvailabilityZones",
					"aws:vpc_subnet:test_app_private1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			az := NewAvailabilityZones()
			CreatePrivateSubnet(appName, "private1", core.IaCValue{Resource: az}, vpc, "0", dag)

			tt.want.Assert(t, dag)
		})
	}
}

func Test_CreatePublicSubnet(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app", "aws:availability_zones:AvailabilityZones", "aws:vpc_subnet:test_app_public1"},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:test_app_public1", Destination: "aws:vpc:test_app"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			az := NewAvailabilityZones()
			CreatePublicSubnet("public1", core.IaCValue{Resource: az}, vpc, "0", dag)

			tt.want.Assert(t, dag)
		})
	}
}

func Test_CreateGatewayVpcEndpoint(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"aws:vpc:test_app", "aws:region:region", "aws:vpc_endpoint:test_app_s3", "aws:vpc_subnet:test_app_1", "aws:vpc_subnet:test_app_2", "aws:route_table:rt"},
				Deps: []coretesting.StringDep{
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:region:region"},
					{Source: "aws:vpc_subnet:test_app_1", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_subnet:test_app_2", Destination: "aws:vpc:test_app"},
					{Source: "aws:vpc_endpoint:test_app_s3", Destination: "aws:route_table:rt"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			subnet1 := NewSubnet("1", vpc, "", PrivateSubnet, core.IaCValue{})
			subnet2 := NewSubnet("2", vpc, "", PrivateSubnet, core.IaCValue{})
			region := NewRegion()
			dag.AddDependency(subnet1, vpc)
			dag.AddDependency(subnet2, vpc)
			CreateGatewayVpcEndpoint("s3", vpc, region, []*RouteTable{&RouteTable{Name: "rt"}}, dag)
			tt.want.Assert(t, dag)
		})
	}
}

func Test_CreateInterfaceVpcEndpoint(t *testing.T) {
	appName := "test-app"
	cases := []struct {
		name string
		want coretesting.ResourcesExpectation
	}{
		{
			name: "happy path",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:test_app",
					"aws:region:region",
					"aws:security_group:test-app",
					"aws:vpc_endpoint:test_app_s3",
					"aws:vpc_subnet:test_app_1",
					"aws:vpc_subnet:test_app_2",
				},
				Deps: []coretesting.StringDep{
					{Destination: "aws:vpc:test_app", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:vpc:test_app", Source: "aws:security_group:test-app"},
					{Destination: "aws:region:region", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:vpc:test_app", Source: "aws:vpc_subnet:test_app_1"},
					{Destination: "aws:vpc:test_app", Source: "aws:vpc_subnet:test_app_2"},
					{Destination: "aws:vpc_subnet:test_app_1", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:vpc_subnet:test_app_2", Source: "aws:vpc_endpoint:test_app_s3"},
					{Destination: "aws:security_group:test-app", Source: "aws:vpc_endpoint:test_app_s3"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dag := core.NewResourceGraph()
			vpc := NewVpc(appName)
			subnet1 := NewSubnet("1", vpc, "", PrivateSubnet, core.IaCValue{})
			subnet2 := NewSubnet("2", vpc, "", PrivateSubnet, core.IaCValue{})
			region := NewRegion()
			dag.AddDependency(subnet1, vpc)
			dag.AddDependency(subnet2, vpc)
			CreateInterfaceVpcEndpoint("s3", vpc, region, dag, &config.Application{AppName: appName})
			tt.want.Assert(t, dag)
		})
	}
}
