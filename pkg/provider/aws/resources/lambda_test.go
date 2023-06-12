package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_LambdaCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
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
				assert.Equal(lambda.ConstructsRef, core.BaseConstructSetOf(eu))
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
				Refs:    core.BaseConstructSetOf(eu),
				Name:    eu.Name,
			}

			tt.Run(t)
		})
	}
}

func Test_LambdaPermissionCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name       string
		permission *LambdaPermission
		paramName  string
		want       coretesting.ResourcesExpectation
	}{
		{
			name: "nil lambda",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:lambda_permission:my_app_permission",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:       "existing lambda",
			permission: &LambdaPermission{Name: "my_app_permission", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:lambda_permission:my_app_permission",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:       "existing lambda no appName",
			paramName:  "my_app_permission",
			permission: &LambdaPermission{Name: "my_app_permission", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:lambda_permission:my_app_permission",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.permission != nil {
				dag.AddResource(tt.permission)
			}

			metadata := LambdaPermissionCreateParams{
				Refs:    core.BaseConstructSetOf(eu2),
				Name:    "permission",
				AppName: "my-app",
			}
			if tt.paramName != "" {
				metadata.AppName = ""
				metadata.Name = tt.paramName
			}
			permission := &LambdaPermission{}
			err := permission.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphLambdaPerm := dag.GetResource(permission.Id())
			permission = graphLambdaPerm.(*LambdaPermission)

			assert.Equal(permission.Name, "my_app_permission")
			if tt.permission == nil {
				assert.Equal(permission.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(permission.BaseConstructsRef(), expect)
			}
		})
	}
}
