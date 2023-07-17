package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_Ec2InstanceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
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
				assert.Equal(instance.ConstructRefs, core.BaseConstructSetOf(eu))
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
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu))
				assert.Equal(instance.ConstructRefs, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Ec2InstanceCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}

func Test_Ec2InstanceMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*Ec2Instance]{
		{
			Name:     "only Ec2Instance",
			Resource: &Ec2Instance{Name: "instance"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ami:my-app-instance",
					"aws:availability_zones:AvailabilityZones",
					"aws:ec2_instance:instance",
					"aws:elastic_ip:my_app_1",
					"aws:iam_instance_profile:my-app-instance",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ec2_instance:instance", Destination: "aws:ami:my-app-instance"},
					{Source: "aws:ec2_instance:instance", Destination: "aws:iam_instance_profile:my-app-instance"},
					{Source: "aws:ec2_instance:instance", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:ec2_instance:instance", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, instance *Ec2Instance) {
				assert.NotNil(instance.AMI)
				assert.NotNil(instance.InstanceProfile)
				assert.NotNil(instance.Subnet)
				assert.NotNil(instance.SecurityGroups)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_AMICreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
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
				assert.Equal(ami.ConstructRefs, core.BaseConstructSetOf(eu))
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
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu))
				assert.Equal(ami.ConstructRefs, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = AMICreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}
