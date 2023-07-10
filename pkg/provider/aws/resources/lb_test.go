package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_LoadBalancerCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[LoadBalancerCreateParams, *LoadBalancer]{
		{
			Name: "nil check ip",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:load_balancer:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, lb *LoadBalancer) {
				assert.Equal(lb.Name, "my-app-instance")
				assert.Equal(lb.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "nil check ip",
			Existing: &LoadBalancer{Name: "my-app-instance", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:load_balancer:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, lb *LoadBalancer) {
				assert.Equal(lb.Name, "my-app-instance")
				assert.Equal(lb.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = LoadBalancerCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "instance",
			}
			tt.Run(t)
		})
	}
}

func Test_LoadBalancerMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*LoadBalancer]{
		{
			Name:     "only lb",
			Resource: &LoadBalancer{Name: "instance"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:internet_gateway:my_app_igw",
					"aws:load_balancer:instance",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
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
					{Source: "aws:load_balancer:instance", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:load_balancer:instance", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:load_balancer:instance", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public0"},
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
			Check: func(assert *assert.Assertions, lb *LoadBalancer) {
				assert.Len(lb.SecurityGroups, 1)
				assert.Len(lb.Subnets, 2)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_TargetGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[TargetGroupCreateParams, *TargetGroup]{
		{
			Name: "nil check ip",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:target_group:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, tg *TargetGroup) {
				assert.Equal(tg.Name, "my-app-instance")
				assert.Equal(tg.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "nil check ip",
			Existing: &TargetGroup{Name: "my-app-instance", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:target_group:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, tg *TargetGroup) {
				assert.Equal(tg.Name, "my-app-instance")
				assert.Equal(tg.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = TargetGroupCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "instance",
			}
			tt.Run(t)
		})
	}
}

func Test_TargetGroupMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*TargetGroup]{
		{
			Name:     "only lb no vpc upstream",
			Resource: &TargetGroup{Name: "instance"},
			AppName:  "my-app",
			WantErr:  true,
		},
		{
			Name:     "lb with upstream vpc",
			Resource: &TargetGroup{Name: "instance"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:target_group:instance", Destination: "aws:vpc:test"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:target_group:instance",
					"aws:vpc:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:target_group:instance", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, tg *TargetGroup) {
				assert.Equal(tg.Vpc.Name, "test")
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_ListenerCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[ListenerCreateParams, *Listener]{
		{
			Name: "nil check ip",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:load_balancer_listener:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, l *Listener) {
				assert.Equal(l.Name, "my-app-instance")
				assert.Equal(l.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "nil check ip",
			Existing: &Listener{Name: "my-app-instance", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:load_balancer_listener:my-app-instance",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, l *Listener) {
				assert.Equal(l.Name, "my-app-instance")
				assert.Equal(l.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = ListenerCreateParams{
				Refs:    core.BaseConstructSetOf(eu),
				AppName: "my-app",
				Name:    "instance",
			}
			tt.Run(t)
		})
	}
}

func Test_ListenerMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*Listener]{
		{
			Name:     "only listener no lb upstream",
			Resource: &Listener{Name: "instance"},
			AppName:  "my-app",
			WantErr:  true,
		},
		{
			Name:     "listener with upstream lb",
			Resource: &Listener{Name: "instance"},
			AppName:  "my-app",
			Existing: []core.Resource{&LoadBalancer{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:load_balancer_listener:instance", Destination: "aws:load_balancer:test"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:load_balancer_listener:instance",
					"aws:load_balancer:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:load_balancer_listener:instance", Destination: "aws:load_balancer:test"},
				},
			},
			Check: func(assert *assert.Assertions, l *Listener) {
				assert.Equal(l.LoadBalancer.Name, "test")
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
