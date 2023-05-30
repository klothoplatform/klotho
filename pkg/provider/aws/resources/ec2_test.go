package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_Ec2InstanceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[Ec2InstanceCreateParams, *Ec2Instance]{
		{
			Name: "nil instance",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ami:my-app-profile",
					"aws:availability_zones:AvailabilityZones",
					"aws:ec2_instance:my-app-profile",
					"aws:elastic_ip:my_app_0",
					"aws:iam_instance_profile:my-app-profile",
					"aws:iam_role:my-app-profile",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:vpc:my_app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ec2_instance:my-app-profile", Destination: "aws:ami:my-app-profile"},
					{Source: "aws:ec2_instance:my-app-profile", Destination: "aws:iam_instance_profile:my-app-profile"},
					{Source: "aws:ec2_instance:my-app-profile", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:ec2_instance:my-app-profile", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:iam_instance_profile:my-app-profile", Destination: "aws:iam_role:my-app-profile"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
				},
			},
			Check: func(assert *assert.Assertions, instance *Ec2Instance) {
				assert.Equal(instance.Name, "my-app-profile")
				assert.NotNil(instance.InstanceProfile)
				assert.NotNil(instance.AMI)
				assert.NotNil(instance.Subnet)
				assert.Len(instance.SecurityGroups, 1)
				assert.Equal(instance.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing instance",
			Existing: &Ec2Instance{Name: "my-app-profile", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ec2_instance:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, instance *Ec2Instance) {
				assert.Equal(instance.Name, "my-app-profile")
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(eu.AnnotationKey))
				assert.Equal(instance.ConstructsRef, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = Ec2InstanceCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(eu.AnnotationKey),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}

func Test_AMICreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
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
				assert.Equal(ami.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing instance",
			Existing: &AMI{Name: "my-app-profile", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ami:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, ami *AMI) {
				assert.Equal(ami.Name, "my-app-profile")
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(eu.AnnotationKey))
				assert.Equal(ami.ConstructsRef, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = AMICreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(eu.AnnotationKey),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}
