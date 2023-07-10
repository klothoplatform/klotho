package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_RdsInstanceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[RdsInstanceCreateParams, *RdsInstance]{
		{
			Name: "nil check ip",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_instance:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, instance *RdsInstance) {
				assert.Equal(instance.Name, "my-app-instance")
				assert.Equal(instance.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "nil check ip",
			Existing: &RdsInstance{Name: "my-app-instance", ConstructRefs: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = RdsInstanceCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "instance",
			}
			tt.Run(t)
		})
	}
}

func Test_RdsInstanceMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*RdsInstance]{
		{
			Name:     "only instance",
			Resource: &RdsInstance{Name: "instance"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:rds_instance:instance",
					"aws:rds_subnet_group:my-app-instance-ubnetroup",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:rds_instance:instance", Destination: "aws:rds_subnet_group:my-app-instance-ubnetroup"},
					{Source: "aws:rds_instance:instance", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:rds_subnet_group:my-app-instance-ubnetroup", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:rds_subnet_group:my-app-instance-ubnetroup", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, instance *RdsInstance) {
				assert.Len(instance.SecurityGroups, 1)
				assert.NotNil(instance.SubnetGroup)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_RdsSubnetGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[RdsSubnetGroupCreateParams, *RdsSubnetGroup]{
		{
			Name: "nil subnet group",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_subnet_group:my-app-sg",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sg *RdsSubnetGroup) {
				assert.Equal(sg.Name, "my-app-sg")
				assert.Equal(sg.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing subnet group",
			Existing: &RdsSubnetGroup{Name: "my-app-sg", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_subnet_group:my-app-sg",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sg *RdsSubnetGroup) {
				assert.Equal(sg.Name, "my-app-sg")
				assert.Equal(sg.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = RdsSubnetGroupCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "sg",
			}
			tt.Run(t)
		})
	}
}

func Test_RdsSubnetGroupMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*RdsSubnetGroup]{
		{
			Name:     "only instance",
			Resource: &RdsSubnetGroup{Name: "sg"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:rds_subnet_group:sg",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:rds_subnet_group:sg", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:rds_subnet_group:sg", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, sg *RdsSubnetGroup) {
				assert.Len(sg.Subnets, 2)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_RdsProxyCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[RdsProxyCreateParams, *RdsProxy]{
		{
			Name: "nil proxy",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy:my-app-proxy",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, proxy *RdsProxy) {
				assert.Equal(proxy.Name, "my-app-proxy")
				assert.Equal(proxy.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing proxy",
			Existing: &RdsProxy{Name: "my-app-proxy", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy:my-app-proxy",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, proxy *RdsProxy) {
				assert.Equal(proxy.Name, "my-app-proxy")
				assert.Equal(proxy.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = RdsProxyCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "proxy",
			}
			tt.Run(t)
		})
	}
}

func Test_RdsProxyMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*RdsProxy]{
		{
			Name:     "only proxy",
			Resource: &RdsProxy{Name: "proxy"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-proxy-ProxyRole",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:rds_proxy:proxy",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:rds_proxy:proxy", Destination: "aws:iam_role:my-app-proxy-ProxyRole"},
					{Source: "aws:rds_proxy:proxy", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:rds_proxy:proxy", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:rds_proxy:proxy", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, proxy *RdsProxy) {
				assert.Len(proxy.Subnets, 2)
				assert.Len(proxy.SecurityGroups, 1)
				assert.NotNil(proxy.Role)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_RdsProxyTargetGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[RdsProxyTargetGroupCreateParams, *RdsProxyTargetGroup]{
		{
			Name: "nil proxy",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy_target_group:my-app-proxy",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, proxy *RdsProxyTargetGroup) {
				assert.Equal(proxy.Name, "my-app-proxy")
				assert.Equal(proxy.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing proxy",
			Existing: &RdsProxyTargetGroup{Name: "my-app-proxy", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_proxy_target_group:my-app-proxy",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, proxy *RdsProxyTargetGroup) {
				assert.Equal(proxy.Name, "my-app-proxy")
				assert.Equal(proxy.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = RdsProxyTargetGroupCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "proxy",
			}
			tt.Run(t)
		})
	}
}

func Test_RdsProxyTargetGroupMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*RdsProxyTargetGroup]{
		{
			Name:     "only proxy tg no instance",
			Resource: &RdsProxyTargetGroup{Name: "proxy", RdsProxy: &RdsProxy{Name: "proxy"}},
			AppName:  "my-app",
			WantErr:  true,
		},
		{
			Name:     "only proxy tg no proxy",
			Resource: &RdsProxyTargetGroup{Name: "proxy", RdsInstance: &RdsInstance{Name: "instance"}},
			AppName:  "my-app",
			WantErr:  true,
		},
		{
			Name:     "functional proxy tg ",
			Resource: &RdsProxyTargetGroup{Name: "proxy", RdsProxy: &RdsProxy{Name: "proxy"}, RdsInstance: &RdsInstance{Name: "instance"}},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rds_instance:instance",
					"aws:rds_proxy:proxy",
					"aws:rds_proxy_target_group:proxy",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:rds_proxy_target_group:proxy", Destination: "aws:rds_instance:instance"},
					{Source: "aws:rds_proxy_target_group:proxy", Destination: "aws:rds_proxy:proxy"},
				},
			},
			Check: func(assert *assert.Assertions, proxy *RdsProxyTargetGroup) {
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
