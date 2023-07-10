package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_SecurityGroupCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[SecurityGroupCreateParams, *SecurityGroup]{
		{
			Name: "nil igw",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.Equal(sg.Name, "my-app")
				assert.Equal(sg.ConstructRefs, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing igw",
			Existing: &SecurityGroup{Name: "my-app", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.Equal(sg.Name, "my-app")
				assert.Equal(sg.ConstructRefs, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = SecurityGroupCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
			}
			tt.Run(t)
		})
	}
}

func Test_SecurityGroupMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*SecurityGroup]{
		{
			Name:     "only sg",
			Resource: &SecurityGroup{Name: "my_app"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:my_app:my_app",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:security_group:my_app:my_app", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.NotNil(sg.Vpc)
			},
		},
		{
			Name:     "sg with downstream vpc",
			Resource: &SecurityGroup{Name: "my_app"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:security_group:my_app", Destination: "aws:vpc:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:test-down:my_app",
					"aws:vpc:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:security_group:test-down:my_app", Destination: "aws:vpc:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.Equal(sg.Vpc.Name, "test-down")
			},
		},
		{
			Name:     "vpc is set ignore downstream",
			Resource: &SecurityGroup{Name: "my_app", Vpc: &Vpc{Name: "test"}},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:security_group:test:my_app", Destination: "aws:vpc:test"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:security_group:test:my_app",
					"aws:vpc:test",
					"aws:vpc:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:security_group:test:my_app", Destination: "aws:vpc:test"},
				},
			},
			Check: func(assert *assert.Assertions, sg *SecurityGroup) {
				assert.Equal(sg.Vpc.Name, "test")
			},
		},
		{
			Name:     "multiple vpcs error",
			Resource: &SecurityGroup{Name: "my_app"},
			AppName:  "my-app",
			Existing: []core.Resource{&Vpc{Name: "test-down"}, &Vpc{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:security_group:my_app", Destination: "aws:vpc:test-down"},
				{Source: "aws:security_group:my_app", Destination: "aws:vpc:test"},
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
