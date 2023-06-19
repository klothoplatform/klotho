package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_VpcCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
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
			metadata := VpcCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
			}

			vpc := &Vpc{}
			err := vpc.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphVpc := dag.GetResource(vpc.Id())
			vpc = graphVpc.(*Vpc)
			assert.Equal(vpc.Name, "my_app")
			if tt.vpc == nil {
				assert.Equal(vpc.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(vpc.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_ElasticIpCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
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
			metadata := EipCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
				IpName:  "ip0",
			}

			eip := &ElasticIp{}
			err := eip.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphEip := dag.GetResource(eip.Id())
			eip = graphEip.(*ElasticIp)
			assert.Equal(eip.Name, "my_app_ip0")
			if tt.eip == nil {
				assert.Equal(eip.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(eip.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_InternetGatewayCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
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
			metadata := IgwCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
			}

			igw := &InternetGateway{}
			err := igw.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphIgw := dag.GetResource(igw.Id())
			igw = graphIgw.(*InternetGateway)
			assert.Equal(igw.Name, "my_app_igw")
			if tt.igw == nil {
				assert.NotNil(igw.Vpc)
				assert.Equal(igw.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(igw.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_NatGatewayCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name string
		nat  *NatGateway
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil nat",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:my_app_public",
					"aws:vpc:my_app",
					"aws:subnet_public:my_app:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing nat",
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
			metadata := NatCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
				AZ:      "0",
			}

			nat := &NatGateway{}
			err := nat.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
			graphNat := dag.GetResource(nat.Id())
			nat = graphNat.(*NatGateway)

			assert.Equal(nat.Name, "my_app_0")
			if tt.nat == nil {
				assert.NotNil(nat.Subnet)
				assert.NotNil(nat.ElasticIp)
				assert.Equal(nat.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(nat.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_SubnetCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name     string
		subnet   *Subnet
		addToDag bool
		want     coretesting.ResourcesExpectation
		wantErr  bool
	}{
		{
			name:     "private subnet az0",
			subnet:   &Subnet{Type: PrivateSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "0"}, CidrBlock: "10.0.0.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_public",
					"aws:vpc:my_app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_public:my_app:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
				},
			},
		},
		{
			name:     "private subnet az1",
			subnet:   &Subnet{Type: PrivateSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}, CidrBlock: "10.0.64.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:vpc:my_app",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
				},
			},
		},
		{
			name:     "public subnet az0",
			subnet:   &Subnet{Type: PublicSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "0"}, CidrBlock: "10.0.128.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:availability_zones:AvailabilityZones",
					"aws:route_table:my_app_public",
					"aws:vpc:my_app",
					"aws:subnet_public:my_app:my_app_public0",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
				},
			},
		},
		{
			name:     "public subnet az1",
			subnet:   &Subnet{Type: PublicSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}, CidrBlock: "10.0.192.0/18"},
			addToDag: false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_app_public",
					"aws:availability_zones:AvailabilityZones",
					"aws:vpc:my_app",
					"aws:subnet_public:my_app:my_app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
				},
			},
		},
		{
			name:     "existing subnet",
			subnet:   &Subnet{Name: "my_app_public0", Type: PublicSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "0"}, ConstructsRef: initialRefs, Vpc: &Vpc{Name: "my_app"}, CidrBlock: "10.0.128.0/18"},
			addToDag: true,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_app_public",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.addToDag {
				dag.AddResource(tt.subnet)
			}
			metadata := SubnetCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
				AZ:      tt.subnet.AvailabilityZone.PropertyVal,
				Type:    tt.subnet.Type,
			}
			subnet := &Subnet{}
			err := subnet.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphSubnet := dag.GetResource(subnet.Id())
			subnet = graphSubnet.(*Subnet)

			assert.Equal(subnet.Name, fmt.Sprintf("my_app_%s%s", tt.subnet.Type, tt.subnet.AvailabilityZone.Property()))
			assert.Equal(subnet.Type, tt.subnet.Type)
			assert.Equal(subnet.AvailabilityZone.Property(), tt.subnet.AvailabilityZone.Property())
			if tt.addToDag == false {
				assert.Equal(subnet.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(subnet.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_RouteTableCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name string
		rt   *RouteTable
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil route table ",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:my_app_private0",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
				},
			},
		},
		{
			name: "existing rt",
			rt:   &RouteTable{Name: "my_app_private0", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:my_app_private0",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
				},
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
			metadata := RouteTableCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
				Name:    "private0",
			}

			rt := &RouteTable{}
			err := rt.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphRT := dag.GetResource(rt.Id())
			rt = graphRT.(*RouteTable)

			assert.Equal(rt.Name, "my_app_private0")
			if tt.rt == nil {
				assert.NotNil(rt.Vpc)
				assert.Equal(rt.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(metadata.Refs)
				assert.Equal(rt.BaseConstructsRef(), expect)
			}
		})
	}
}
