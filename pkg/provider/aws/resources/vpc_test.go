package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_VpcCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[VpcCreateParams, *Vpc]{
		{
			Name: "nil vpc",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, vpc *Vpc) {
				assert.Equal(vpc.Name, "my_app")
				assert.Equal(vpc.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing load balancer",
			Existing: &Vpc{Name: "my_app", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, vpc *Vpc) {
				assert.Equal(vpc.Name, "my_app")
				assert.Equal(vpc.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = VpcCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
			}
			tt.Run(t)
		})
	}
}

func Test_ElasticIpCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[EipCreateParams, *ElasticIp]{
		{
			Name: "nil vpc",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:my_app_ip0",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, eip *ElasticIp) {
				assert.Equal(eip.Name, "my_app_ip0")
				assert.Equal(eip.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing load balancer",
			Existing: &ElasticIp{Name: "my_app_ip0", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:my_app_ip0",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, eip *ElasticIp) {
				assert.Equal(eip.Name, "my_app_ip0")
				assert.Equal(eip.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = EipCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "ip0",
			}
			tt.Run(t)
		})
	}
}

func Test_InternetGatewayCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[IgwCreateParams, *InternetGateway]{
		{
			Name: "nil igw",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, eip *InternetGateway) {
				assert.Equal(eip.Name, "my_app_igw")
				assert.Equal(eip.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing igw",
			Existing: &InternetGateway{Name: "my_app_igw", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, eip *InternetGateway) {
				assert.Equal(eip.Name, "my_app_igw")
				assert.Equal(eip.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = IgwCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
			}
			tt.Run(t)
		})
	}
}

func Test_InternetGatewayMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*InternetGateway]{
		{
			Name:     "only igw",
			Resource: &InternetGateway{Name: "my_app_igw"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, eip *InternetGateway) {
				assert.NotNil(eip.Vpc)
			},
		},
		{
			Name:     "igw with downstream vpc",
			Resource: &InternetGateway{Name: "my_app_igw"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:vpc:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, eip *InternetGateway) {
				assert.Equal(eip.Vpc.Name, "test-down")
			},
		},
		{
			Name:     "vpc is set ignore upstream",
			Resource: &InternetGateway{Name: "my_app_igw", Vpc: &Vpc{Name: "test"}},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:vpc:test",
					"aws:vpc:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test-down"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, eip *InternetGateway) {
				assert.Equal(eip.Vpc.Name, "test")
			},
		},
		{
			Name:     "multiple vpcs error",
			Resource: &InternetGateway{Name: "my_app_igw"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test-down"},
				{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_NatGatewayCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[NatCreateParams, *NatGateway]{
		{
			Name: "nil nat",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:nat_gateway:my_app_nat",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, nat *NatGateway) {
				assert.Equal(nat.Name, "my_app_nat")
				assert.Equal(nat.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing nat",
			Existing: &NatGateway{Name: "my_app_nat", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:nat_gateway:my_app_nat",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, nat *NatGateway) {
				assert.Equal(nat.Name, "my_app_nat")
				assert.Equal(nat.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = NatCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "nat",
			}
			tt.Run(t)
		})
	}
}

func Test_NatGatewayMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*NatGateway]{
		{
			Name:     "only nat",
			Resource: &NatGateway{Name: "my_app_nat"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_nat",
					"aws:route_table:my_app_public",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:availability_zones:AvailabilityZones",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
				},
			},
			Check: func(assert *assert.Assertions, nat *NatGateway) {
				assert.Equal(nat.ElasticIp.Name, "my_app_1")
				assert.Equal(nat.Subnet.Name, "my_app_public1")
			},
		},
		{
			Name:     "nat with downstream vpc",
			Resource: &NatGateway{Name: "my_app_nat"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:vpc:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_nat",
					"aws:route_table:my_app_public",
					"aws:subnet_public:test-down:my_app_public1",
					"aws:availability_zones:AvailabilityZones",
					"aws:vpc:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test-down"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_public:test-down:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:vpc:test-down"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test-down:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:test-down"},
					{Source: "aws:subnet_public:test-down:my_app_public1", Destination: "aws:vpc:test-down"},
					{Source: "aws:subnet_public:test-down:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
				},
			},
			Check: func(assert *assert.Assertions, nat *NatGateway) {
				assert.Equal(nat.ElasticIp.Name, "my_app_1")
				assert.Equal(nat.Subnet.Name, "my_app_public1")
				assert.Equal(nat.Subnet.Vpc.Name, "test-down")
			},
		},
		{
			Name:     "nat with downstream subnet and eip",
			Resource: &NatGateway{Name: "my_app_nat"},
			AppName:  "my-app",
			Existing: []core.Resource{&Subnet{Name: "test-down", Type: PublicSubnet}, &ElasticIp{Name: "test_1"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_public:test-down"},
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:test_1"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:test_1",
					"aws:nat_gateway:my_app_nat",
					"aws:subnet_public:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:test_1"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_public:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, nat *NatGateway) {
				assert.Equal(nat.ElasticIp.Name, "test_1")
				assert.Equal(nat.Subnet.Name, "test-down")
			},
		},
		{
			Name:     "nat with subnet and eip",
			Resource: &NatGateway{Name: "my_app_nat", ElasticIp: &ElasticIp{Name: "test_1"}, Subnet: &Subnet{Name: "test-down", Type: PublicSubnet}},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:elastic_ip:test_1",
					"aws:nat_gateway:my_app_nat",
					"aws:subnet_public:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:test_1"},
					{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_public:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, nat *NatGateway) {
				assert.Equal(nat.ElasticIp.Name, "test_1")
				assert.Equal(nat.Subnet.Name, "test-down")
			},
		},
		{
			Name:     "multiple vpcs error",
			Resource: &NatGateway{Name: "my_app_nat"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:vpc:test-down"},
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:vpc:test"},
			},
			WantErr: true,
		},
		{
			Name:     "multiple subnets error",
			Resource: &NatGateway{Name: "my_app_nat"},
			AppName:  "my-app",
			Existing: []core.Resource{&Subnet{Name: "test-down"}, &Subnet{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_:test-down"},
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:subnet_:test"},
			},
			WantErr: true,
		},
		{
			Name:     "multiple eips error",
			Resource: &NatGateway{Name: "my_app_nat"},
			AppName:  "my-app",
			Existing: []core.Resource{&ElasticIp{Name: "test-down"}, &ElasticIp{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:test-down"},
				{Source: "aws:nat_gateway:my_app_nat", Destination: "aws:elastic_ip:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_SubnetCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[SubnetCreateParams, *Subnet]{
		{
			Name: "nil subnet",
			Params: SubnetCreateParams{
				AZ:   "0",
				Type: PublicSubnet,
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:subnet_public:my_app_public0",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Name, "my_app_public0")
				assert.Equal(subnet.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name: "nil subnet no az",
			Params: SubnetCreateParams{
				Type: PublicSubnet,
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:subnet_public:my_app_public",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Name, "my_app_public")
				assert.Equal(subnet.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name: "nil subnet no Type",
			Params: SubnetCreateParams{
				AZ: "0",
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:subnet_:my_app_0",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Name, "my_app_0")
				assert.Equal(subnet.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name: "nil subnet no Type or az",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:subnet_:my_app_",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Name, "my_app_")
				assert.Equal(subnet.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing subnet",
			Existing: &Subnet{Name: "my_app_", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:subnet_:my_app_",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Name, "my_app_")
				assert.Equal(subnet.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = SubnetCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				AZ:      tt.Params.AZ,
				Type:    tt.Params.Type,
			}
			tt.Run(t)
		})
	}
}

func Test_SubnetMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*Subnet]{
		{
			Name:     "only Subnet",
			Resource: &Subnet{Name: "my_app_subnet"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_app_public",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Vpc.Name, "my_app")
				assert.Equal(subnet.AvailabilityZone.PropertyVal, "1")
				assert.Equal(subnet.Type, PublicSubnet)
			},
		},
		{
			Name:     "subnet has vpc upstream and should assign itself private",
			Resource: &Subnet{Name: "my_app_subnet", AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}},
			Existing: []core.Resource{&Vpc{Name: "test"}, &Subnet{Name: "test-down", Type: PublicSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:subnet_public:test-down", Destination: "aws:vpc:test"},
				{Source: "aws:subnet_:my_app_subnet", Destination: "aws:vpc:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:subnet_private:test:my_app_private1",
					"aws:subnet_public:test-down",
					"aws:subnet_public:test:my_app_public1",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:test:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:test:my_app_private1", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test-down", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Vpc.Name, "test")
				assert.Equal(subnet.AvailabilityZone.PropertyVal, "1")
				assert.Equal(subnet.Type, PrivateSubnet)
			},
		},
		{
			Name:     "subnet has vpc tied to it and should assign itself az of 0",
			Resource: &Subnet{Name: "my_app_subnet", Type: PublicSubnet, Vpc: &Vpc{Name: "test"}},
			Existing: []core.Resource{&Vpc{Name: "test"}, &Subnet{Name: "test-down", Type: PublicSubnet, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:subnet_public:test-down", Destination: "aws:vpc:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_app_public",
					"aws:subnet_public:test-down",
					"aws:subnet_public:test:my_app_public0",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test-down", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_public:test:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public0", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, subnet *Subnet) {
				assert.Equal(subnet.Vpc.Name, "test")
				assert.Equal(subnet.AvailabilityZone.PropertyVal, "0")
				assert.Equal(subnet.Type, PublicSubnet)
			},
		},
		{
			Name:     "multiple vpcs error",
			Resource: &Subnet{Name: "subnet", Type: PublicSubnet},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:subnet_public:subnet", Destination: "aws:vpc:test-down"},
				{Source: "aws:subnet_public:subnet", Destination: "aws:vpc:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_RouteTableCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[RouteTableCreateParams, *RouteTable]{
		{
			Name: "nil route table",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:my_app_rt",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Name, "my_app_rt")
				assert.Equal(rt.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing route table",
			Existing: &RouteTable{Name: "my_app_rt", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:my_app_rt",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Name, "my_app_rt")
				assert.Equal(rt.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = RouteTableCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "rt",
			}
			tt.Run(t)
		})
	}
}

func Test_RouteTableMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*RouteTable]{
		{
			Name:     "only route table",
			Resource: &RouteTable{Name: "my_rt"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:my_rt",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:route_table:my_rt", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Vpc.Name, "my_app")
			},
		},
		{
			Name:     "route table with vpc upstream",
			Resource: &RouteTable{Name: "my_rt"},
			Existing: []core.Resource{&Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:route_table:my_rt",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Vpc.Name, "test")
			},
		},
		{
			Name:     "route table with private subnets upstream",
			Resource: &RouteTable{Name: "my_rt"},
			Existing: []core.Resource{&Subnet{Name: "test", Type: PrivateSubnet, Vpc: &Vpc{Name: "test"}, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_private:test:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_public",
					"aws:route_table:my_rt",
					"aws:subnet_private:test:test",
					"aws:subnet_public:test:my_app_public1",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:route_table:my_rt", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_rt", Destination: "aws:subnet_private:test:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:test:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:test"},
					{Source: "aws:subnet_private:test:test", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:test:my_app_public1", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Vpc.Name, "test")
			},
		},
		{
			Name:     "route table with private subnet and nat upstream",
			Resource: &RouteTable{Name: "my_rt"},
			Existing: []core.Resource{
				&Vpc{Name: "test"},
				&Subnet{Name: "test", Type: PrivateSubnet, Vpc: &Vpc{Name: "test"}, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}},
				&NatGateway{Name: "mynat", Subnet: &Subnet{Name: "test", Type: PrivateSubnet, Vpc: &Vpc{Name: "test"}, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}}},
			},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_private:test:test"},
				{Source: "aws:nat_gateway:mynat", Destination: "aws:vpc:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:nat_gateway:mynat",
					"aws:route_table:my_rt",
					"aws:subnet_private:test:test",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:nat_gateway:mynat", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:nat_gateway:mynat"},
					{Source: "aws:route_table:my_rt", Destination: "aws:subnet_private:test:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Vpc.Name, "test")
			},
		},
		{
			Name:     "route table with Public subnets upstream",
			Resource: &RouteTable{Name: "my_rt"},
			Existing: []core.Resource{&Subnet{Name: "test", Type: PublicSubnet, Vpc: &Vpc{Name: "test"}, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:my_app_igw",
					"aws:route_table:my_rt",
					"aws:subnet_public:test:test",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Vpc.Name, "test")
			},
		},
		{
			Name:     "route table with Public subnet and internet gateway upstream",
			Resource: &RouteTable{Name: "my_rt"},
			Existing: []core.Resource{
				&Vpc{Name: "test"},
				&Subnet{Name: "test", Type: PublicSubnet, Vpc: &Vpc{Name: "test"}, AvailabilityZone: &AwsResourceValue{PropertyVal: "1"}},
				&InternetGateway{Name: "myigw"},
			},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test:test"},
				{Source: "aws:internet_gateway:myigw", Destination: "aws:vpc:test"},
			},
			AppName: "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:internet_gateway:myigw",
					"aws:route_table:my_rt",
					"aws:subnet_public:test:test",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:myigw", Destination: "aws:vpc:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:internet_gateway:myigw"},
					{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test:test"},
					{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, rt *RouteTable) {
				assert.Equal(rt.Vpc.Name, "test")
			},
		},
		{
			Name:     "multiple vpcs error",
			Resource: &RouteTable{Name: "my_rt"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test-down"},
				{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test"},
			},
			WantErr: true,
		},
		{
			Name:     "multiple vpcs from subnets error",
			Resource: &RouteTable{Name: "my_rt"},
			AppName:  "my-app",
			Existing: []core.Resource{
				&Vpc{Name: "test-down"},
				&Subnet{Name: "test", Vpc: &Vpc{Name: "test"}, Type: PublicSubnet},
				&Subnet{Name: "test2", Vpc: &Vpc{Name: "test2"}, Type: PublicSubnet},
			},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test-down"},
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test:test"},
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test2:test2"},
			},
			WantErr: true,
		},
		{
			Name:     "multiple vpcs from conflicting vpc and subnets error",
			Resource: &RouteTable{Name: "my_rt"},
			AppName:  "my-app",
			Existing: []core.Resource{
				&Vpc{Name: "test-down"},
				&Subnet{Name: "test", Vpc: &Vpc{Name: "test"}, Type: PublicSubnet},
			},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:route_table:my_rt", Destination: "aws:vpc:test-down"},
				{Source: "aws:route_table:my_rt", Destination: "aws:subnet_public:test:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
