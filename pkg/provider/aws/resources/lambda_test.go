package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_LambdaCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}

	cases := []coretesting.CreateCase[LambdaCreateParams, *LambdaFunction]{
		{
			Name: "nil lambda",
			Want: coretesting.ResourcesExpectation{
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
			Check: func(assert *assert.Assertions, lambda *LambdaFunction) {
				assert.Equal(lambda.Name, "my-app-test")
				assert.ElementsMatch(lambda.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
			},
		},
		{
			Name:     "existing lambda",
			Existing: &LambdaFunction{Name: "my-app-test"},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = LambdaCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{eu.AnnotationKey},
				Name:    eu.ID,
			}

			tt.Run(t)
		})
	}
}
