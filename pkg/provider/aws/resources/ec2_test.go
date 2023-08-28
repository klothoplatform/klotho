package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_Ec2InstanceCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[Ec2InstanceCreateParams, *Ec2Instance]{
		{
			Name: "nil instance",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ec2_instance:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, instance *Ec2Instance) {
				assert.Equal(instance.Name, "my-app-profile")
				assert.Equal(instance.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing instance",
			Existing: &Ec2Instance{Name: "my-app-profile", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ec2_instance:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, instance *Ec2Instance) {
				assert.Equal(instance.Name, "my-app-profile")
				expect := initialRefs.CloneWith(construct.BaseConstructSetOf(eu))
				assert.Equal(instance.ConstructRefs, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Ec2InstanceCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}

func Test_AMICreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[AMICreateParams, *AMI]{
		{
			Name: "nil instance",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ami:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ami *AMI) {
				assert.Equal(ami.Name, "my-app-profile")
				assert.Equal(ami.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing instance",
			Existing: &AMI{Name: "my-app-profile", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ami:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ami *AMI) {
				assert.Equal(ami.Name, "my-app-profile")
				expect := initialRefs.CloneWith(construct.BaseConstructSetOf(eu))
				assert.Equal(ami.ConstructRefs, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = AMICreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}
