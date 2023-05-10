package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_LambdaCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}, DockerfilePath: "path"}
	cases := []struct {
		name    string
		lambda  *LambdaFunction
		vpc     bool
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil lambda",
			vpc:  false,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-test",
					"aws:ecr_repo:my-app",
					"aws:iam_role:my-app-test-ExecutionRole",
					"aws:lambda_function:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
				},
			},
		},
		{
			name:    "existing lambda",
			lambda:  &LambdaFunction{Name: "my-app-test"},
			vpc:     false,
			wantErr: true,
		},
		{
			name: "nil lambda with vpc",
			vpc:  true,
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:ecr_image:my-app-test",
					"aws:ecr_repo:my-app",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-test-ExecutionRole",
					"aws:internet_gateway:my_app_igw",
					"aws:lambda_function:my-app-test",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_0",
					"aws:route_table:my_app_1",
					"aws:route_table:my_app_igw",
					"aws:security_group:my-app",
					"aws:vpc:my_app",
					"aws:vpc_subnet:my_app_private0",
					"aws:vpc_subnet:my_app_private1",
					"aws:vpc_subnet:my_app_public0",
					"aws:vpc_subnet:my_app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:security_group:my-app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:vpc_subnet:my_app_private0"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:vpc_subnet:my_app_private1"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:vpc_subnet:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:vpc_subnet:my_app_public1"},
					{Source: "aws:route_table:my_app_0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:vpc_subnet:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:my_app_public1", Destination: "aws:vpc:my_app"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.lambda != nil {
				dag.AddResource(tt.lambda)
			}

			metadata := LambdaCreateParams{
				AppName:          "my-app",
				Unit:             eu,
				NetworkPlacement: "private",
				Vpc:              tt.vpc,
				Params:           config.ServerlessTypeParams{Timeout: 60, Memory: 250},
			}
			lambda := &LambdaFunction{}
			err := lambda.Create(dag, metadata)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			assert.Equal(lambda.Name, "my-app-test")
			assert.Equal(lambda.Image.Dockerfile, fmt.Sprintf("./%s/%s", eu.ID, eu.DockerfilePath))
			assert.Equal(lambda.Role.AssumeRolePolicyDoc, LAMBDA_ASSUMER_ROLE_POLICY)
			assert.Equal(lambda.Role.Name, "my-app-test-ExecutionRole")
			assert.Equal(lambda.MemorySize, 250)
			assert.Equal(lambda.Timeout, 60)
			assert.ElementsMatch(lambda.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
		})
	}
}
