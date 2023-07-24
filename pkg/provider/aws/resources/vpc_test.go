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
