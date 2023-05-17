package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
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
					"aws:log_group:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test", Destination: "aws:ecr_repo:my-app"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:ecr_image:my-app-test"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:iam_role:my-app-test-ExecutionRole"},
					{Source: "aws:lambda_function:my-app-test", Destination: "aws:log_group:my-app-test"},
				},
			},
		},
		{
			name:    "existing lambda",
			lambda:  &LambdaFunction{Name: "my-app-test"},
			vpc:     false,
			wantErr: true,
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
				AppName: "my-app",
				Refs:    []core.AnnotationKey{eu.AnnotationKey},
				Name:    eu.ID,
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
			assert.ElementsMatch(lambda.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
		})
	}
}
