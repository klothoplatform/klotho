package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_RestApiCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name    string
		api     *RestApi
		apiName string
		want    coretesting.ResourcesExpectation
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rest_api:my-app-rest-api",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name: "existing repo",
			api:  &RestApi{Name: "my-app-rest-api", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rest_api:my-app-rest-api",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:    "existing repo just name params",
			apiName: "my-app-rest-api",
			api:     &RestApi{Name: "my-app-rest-api", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:rest_api:my-app-rest-api",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.api != nil {
				dag.AddResource(tt.api)
			}
			metadata := RestApiCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
				Name:    "rest-api",
			}
			if tt.apiName != "" {
				metadata.AppName = ""
				metadata.Name = tt.apiName
			}

			api := &RestApi{}
			err := api.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)

			graphApi := dag.GetResource(api.Id())
			api = graphApi.(*RestApi)

			assert.Equal(api.Name, "my-app-rest-api")
			if tt.api == nil {
				assert.Equal(api.ConstructsRef, metadata.Refs)
			} else {
				initialRefs.Add(&core.ExecutionUnit{Name: "test"})
				assert.Equal(api.BaseConstructsRef(), initialRefs)
			}
		})
	}
}

func Test_RestApiConfigure(t *testing.T) {
	cases := []struct {
		name   string
		params RestApiConfigureParams
		want   *RestApi
	}{
		{
			name: "filled params",
			params: RestApiConfigureParams{
				BinaryMediaTypes: []string{"test"},
			},
			want: &RestApi{BinaryMediaTypes: []string{"test"}},
		},
		{
			name:   "defaults",
			params: RestApiConfigureParams{},
			want:   &RestApi{BinaryMediaTypes: []string{"application/octet-stream", "image/*"}},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			api := &RestApi{}
			err := api.Configure(tt.params)

			if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want, api)
		})
	}
}

func Test_ApiResourceCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []coretesting.CreateCase[ApiResourceCreateParams, *ApiResource]{
		{
			Name: "nil resource",
			Params: ApiResourceCreateParams{
				Path: "/my/api/route",
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_resource:my-app-/my",
					"aws:api_resource:my-app-/my/api",
					"aws:api_resource:my-app-/my/api/route",
					"aws:rest_api:my-api",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/api", Destination: "aws:api_resource:my-app-/my"},
					{Source: "aws:api_resource:my-app-/my/api", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/api/route", Destination: "aws:api_resource:my-app-/my/api"},
					{Source: "aws:api_resource:my-app-/my/api/route", Destination: "aws:rest_api:my-api"},
				},
			},
			Check: func(assert *assert.Assertions, resource *ApiResource) {
				assert.Equal("my-app-/my/api/route", resource.Name)
				assert.Equal(resource.PathPart, "route")
				assert.Equal(resource.ConstructsRef, core.BaseConstructSetOf(eu2))
			},
		},
		{
			Name: "existing resource",
			Params: ApiResourceCreateParams{
				Path: "/my/api/route",
			},
			Existing: &ApiResource{Name: "my-app-/my/api/route", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_resource:my-app-/my/api/route",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, resource *ApiResource) {
				assert.Equal("my-app-/my/api/route", resource.Name)
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(resource.BaseConstructsRef(), expect)

			},
		},
		{
			Name: "path param nil resource",
			Params: ApiResourceCreateParams{
				Path: "/my/:api/route/:method",
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_resource:my-app-/my",
					"aws:api_resource:my-app-/my/-api",
					"aws:api_resource:my-app-/my/-api/route",
					"aws:api_resource:my-app-/my/-api/route/-method",
					"aws:rest_api:my-api",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/-api", Destination: "aws:api_resource:my-app-/my"},
					{Source: "aws:api_resource:my-app-/my/-api", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/-api/route", Destination: "aws:api_resource:my-app-/my/-api"},
					{Source: "aws:api_resource:my-app-/my/-api/route", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/-api/route/-method", Destination: "aws:api_resource:my-app-/my/-api/route"},
					{Source: "aws:api_resource:my-app-/my/-api/route/-method", Destination: "aws:rest_api:my-api"},
				},
			},
			Check: func(assert *assert.Assertions, resource *ApiResource) {
				assert.Equal("my-app-/my/-api/route/-method", resource.Name)
				assert.Equal(resource.PathPart, "{method}")
				assert.Equal(resource.ConstructsRef, core.BaseConstructSetOf(eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params.AppName = "my-app"
			tt.Params.Refs = core.BaseConstructSetOf(eu2)
			tt.Params.ApiName = "my-api"
			tt.Run(t)
		})
	}
}

func Test_ApiIntegrationCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name        string
		integration *ApiIntegration
		want        coretesting.ResourcesExpectation
		wantErr     bool
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_integration:my-app-/my/api/route-post",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.integration != nil {
				dag.AddResource(tt.integration)
			}
			metadata := ApiIntegrationCreateParams{
				AppName:    "my-app",
				Refs:       core.BaseConstructSetOf(eu2),
				Path:       "/my/api/route",
				ApiName:    "my-api",
				HttpMethod: "post",
			}

			integration := &ApiIntegration{}
			err := integration.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)

			graphIntegration := dag.GetResource(integration.Id())
			integration = graphIntegration.(*ApiIntegration)

			assert.Equal(integration.Name, "my-app-/my/api/route-post")
			if tt.integration == nil {
				assert.Equal(integration.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(integration.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_ApiMethodCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name   string
		method *ApiMethod
		want   coretesting.ResourcesExpectation
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_method:my-app-/my/api/route-post",
					"aws:api_resource:my-app-/my",
					"aws:api_resource:my-app-/my/api",
					"aws:api_resource:my-app-/my/api/route",
					"aws:rest_api:my-api",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_method:my-app-/my/api/route-post", Destination: "aws:api_resource:my-app-/my/api/route"},
					{Source: "aws:api_method:my-app-/my/api/route-post", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/api", Destination: "aws:api_resource:my-app-/my"},
					{Source: "aws:api_resource:my-app-/my/api", Destination: "aws:rest_api:my-api"},
					{Source: "aws:api_resource:my-app-/my/api/route", Destination: "aws:api_resource:my-app-/my/api"},
					{Source: "aws:api_resource:my-app-/my/api/route", Destination: "aws:rest_api:my-api"},
				},
			},
		},
		{
			name:   "existing repo",
			method: &ApiMethod{Name: "my-app-/my/api/route-post", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_method:my-app-/my/api/route-post",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.method != nil {
				dag.AddResource(tt.method)
			}
			metadata := ApiMethodCreateParams{
				AppName:    "my-app",
				Refs:       core.BaseConstructSetOf(eu2),
				Path:       "/my/api/route",
				ApiName:    "my-api",
				HttpMethod: "post",
			}

			method := &ApiMethod{}
			err := method.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphMethod := dag.GetResource(method.Id())
			method = graphMethod.(*ApiMethod)

			assert.Equal(method.Name, "my-app-/my/api/route-post")
			if tt.method == nil {
				assert.NotNil(method.RestApi)
				assert.NotNil(method.Resource)
				assert.Equal(method.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(method.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_ApiMethodConfigure(t *testing.T) {
	cases := []struct {
		name   string
		params ApiMethodConfigureParams
		want   *ApiMethod
	}{
		{
			name: "filled params",
			params: ApiMethodConfigureParams{
				Authorization: "IAM",
			},
			want: &ApiMethod{Authorization: "IAM"},
		},
		{
			name:   "defaults",
			params: ApiMethodConfigureParams{},
			want:   &ApiMethod{Authorization: "None"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			method := &ApiMethod{}
			err := method.Configure(tt.params)

			if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want, method)
		})
	}
}

func Test_ApiDeploymentCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name       string
		deployment *ApiDeployment
		want       coretesting.ResourcesExpectation
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_deployment:my-app-deployment",
					"aws:rest_api:my-app-deployment",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_deployment:my-app-deployment", Destination: "aws:rest_api:my-app-deployment"},
				},
			},
		},
		{
			name:       "existing repo",
			deployment: &ApiDeployment{Name: "my-app-deployment", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_deployment:my-app-deployment",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.deployment != nil {
				dag.AddResource(tt.deployment)
			}
			metadata := ApiDeploymentCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu2),
				Name:    "deployment",
			}

			deployment := &ApiDeployment{}
			err := deployment.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphDeployment := dag.GetResource(deployment.Id())
			deployment = graphDeployment.(*ApiDeployment)

			assert.Equal(deployment.Name, "my-app-deployment")
			if tt.deployment == nil {
				assert.NotNil(deployment.RestApi)
				assert.Equal(deployment.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(deployment.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_ApiStageCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name  string
		stage *ApiStage
		want  coretesting.ResourcesExpectation
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_deployment:my-app-stage",
					"aws:api_stage:my-app-stage",
					"aws:rest_api:my-app-stage",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_deployment:my-app-stage", Destination: "aws:rest_api:my-app-stage"},
					{Source: "aws:api_stage:my-app-stage", Destination: "aws:api_deployment:my-app-stage"},
					{Source: "aws:api_stage:my-app-stage", Destination: "aws:rest_api:my-app-stage"},
				},
			},
		},
		{
			name:  "existing repo",
			stage: &ApiStage{Name: "my-app-stage", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_stage:my-app-stage",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.stage != nil {
				dag.AddResource(tt.stage)
			}
			metadata := ApiStageCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu2),
				Name:    "stage",
			}

			stage := &ApiStage{}
			err := stage.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphStage := dag.GetResource(stage.Id())
			stage = graphStage.(*ApiStage)

			assert.Equal(stage.Name, "my-app-stage")
			if tt.stage == nil {
				assert.NotNil(stage.RestApi)
				assert.Equal(stage.ConstructsRef, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(stage.BaseConstructsRef(), expect)
			}
		})
	}
}
func Test_ApiStageConfigure(t *testing.T) {
	cases := []struct {
		name   string
		params ApiStageConfigureParams
		want   *ApiStage
	}{
		{
			name: "filled params",
			params: ApiStageConfigureParams{
				StageName: "production",
			},
			want: &ApiStage{StageName: "production"},
		},
		{
			name:   "defaults",
			params: ApiStageConfigureParams{},
			want:   &ApiStage{StageName: "stage"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			stage := &ApiStage{}
			err := stage.Configure(tt.params)

			if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want, stage)
		})
	}
}
