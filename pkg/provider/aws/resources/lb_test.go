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
				{Source: "aws:load_balancer:test", Destination: "aws:load_balancer_listener:instance"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:load_balancer_listener:instance",
					"aws:load_balancer:test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:load_balancer:test", Destination: "aws:load_balancer_listener:instance"},
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
