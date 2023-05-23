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
	cases := []struct {
		name    string
		lambda  *LambdaFunction
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil lambda",
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

			graphLambda := dag.GetResource(lambda.Id())
			lambda = graphLambda.(*LambdaFunction)

			assert.Equal(lambda.Name, "my-app-test")
			assert.ElementsMatch(lambda.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
		})
	}
}

func Test_LambdaPermissionCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
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
				Refs:    []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
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
				assert.ElementsMatch(permission.ConstructsRef, metadata.Refs)
			} else {
				assert.ElementsMatch(permission.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))
			}
		})
	}
}
