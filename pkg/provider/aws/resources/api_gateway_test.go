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
			api:  &RestApi{Name: "my-app-rest-api", ConstructRefs: initialRefs},
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
			api:     &RestApi{Name: "my-app-rest-api", ConstructRefs: initialRefs},
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
				assert.Equal(api.ConstructRefs, metadata.Refs)
			} else {
				initialRefs.Add(&core.ExecutionUnit{Name: "test"})
				assert.Equal(api.BaseConstructRefs(), initialRefs)
			}
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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:api_resource:my-app-/my/api"},
					{Source: "aws:api_resource:my-app-/my/api", Destination: "aws:api_resource:my-app-/my/api/route"},
				},
			},
			Check: func(assert *assert.Assertions, resource *ApiResource) {
				assert.Equal("my-app-/my/api/route", resource.Name)
				assert.Equal(resource.PathPart, "route")
				assert.Equal(resource.ConstructRefs, core.BaseConstructSetOf(eu2))
			},
		},
		{
			Name: "existing resource",
			Params: ApiResourceCreateParams{
				Path: "/my/api/route",
			},
			Existing: &ApiResource{Name: "my-app-/my/api/route", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_resource:my-app-/my/api/route",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, resource *ApiResource) {
				assert.Equal("my-app-/my/api/route", resource.Name)
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(resource.BaseConstructRefs(), expect)

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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:api_resource:my-app-/my/-api"},
					{Source: "aws:api_resource:my-app-/my/-api", Destination: "aws:api_resource:my-app-/my/-api/route"},
					{Source: "aws:api_resource:my-app-/my/-api/route", Destination: "aws:api_resource:my-app-/my/-api/route/-method"},
				},
			},
			Check: func(assert *assert.Assertions, resource *ApiResource) {
				assert.Equal("my-app-/my/-api/route/-method", resource.Name)
				assert.Equal(resource.PathPart, "{method}")
				assert.Equal(resource.ConstructRefs, core.BaseConstructSetOf(eu2))
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
				assert.Equal(integration.ConstructRefs, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(integration.BaseConstructRefs(), expect)
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
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:api_resource:my-app-/my/api"},
					{Source: "aws:api_resource:my-app-/my/api", Destination: "aws:api_resource:my-app-/my/api/route"},
					{Source: "aws:api_resource:my-app-/my/api/route", Destination: "aws:api_method:my-app-/my/api/route-post"},
				},
			},
		},
		{
			name:   "existing repo",
			method: &ApiMethod{Name: "my-app-/my/api/route-post", ConstructRefs: initialRefs},
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
				assert.NotNil(method.Resource)
				assert.Equal(method.ConstructRefs, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(method.BaseConstructRefs(), expect)
			}
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
			name: "nil deployment",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_deployment:my-app-deployment",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:       "existing deployment",
			deployment: &ApiDeployment{Name: "my-app-deployment", ConstructRefs: initialRefs},
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
				assert.Equal(deployment.ConstructRefs, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(deployment.BaseConstructRefs(), expect)
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
					"aws:api_stage:my-app-stage",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:  "existing repo",
			stage: &ApiStage{Name: "my-app-stage", ConstructRefs: initialRefs},
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
				assert.Equal(stage.ConstructRefs, metadata.Refs)
			} else {
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(stage.BaseConstructRefs(), expect)
			}
		})
	}
}
