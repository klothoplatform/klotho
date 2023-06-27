package knowledgebase

import (
	"testing"

	dgraph "github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ExpandEdges(t *testing.T) {
	cases := []struct {
		name string
		edge graph.Edge[core.Resource]
		want coretesting.ResourcesExpectation
	}{
		{
			name: "single expose lambda",
			edge: graph.Edge[core.Resource]{
				Source:      &resources.RestApi{Name: "api"},
				Destination: &resources.LambdaFunction{Name: "lambda"},
				Properties: dgraph.EdgeProperties{
					Data: knowledgebase.EdgeData{
						AppName: "my-app",
						Routes:  []core.Route{{Path: "/my/route/1", Verb: "post"}, {Path: "/my/route/1", Verb: "get"}, {Path: "/my/route/2", Verb: "post"}, {Path: "/", Verb: "get"}},
					},
				},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:api_integration:my-app-/-GET",
					"aws:api_integration:my-app-/my/route/1-GET",
					"aws:api_integration:my-app-/my/route/1-POST",
					"aws:api_integration:my-app-/my/route/2-POST",
					"aws:api_method:my-app-/-GET",
					"aws:api_method:my-app-/my/route/1-GET",
					"aws:api_method:my-app-/my/route/1-POST",
					"aws:api_method:my-app-/my/route/2-POST",
					"aws:api_resource:my-app-/my",
					"aws:api_resource:my-app-/my/route",
					"aws:api_resource:my-app-/my/route/1",
					"aws:api_resource:my-app-/my/route/2",
					"aws:lambda_function:lambda",
					"aws:lambda_permission:lambda_awsrest_apiapi",
					"aws:rest_api:api",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_integration:my-app-/-GET", Destination: "aws:api_method:my-app-/-GET"},
					{Source: "aws:api_integration:my-app-/-GET", Destination: "aws:lambda_function:lambda"},
					{Source: "aws:api_integration:my-app-/-GET", Destination: "aws:rest_api:api"},
					{Source: "aws:api_integration:my-app-/my/route/1-GET", Destination: "aws:api_method:my-app-/my/route/1-GET"},
					{Source: "aws:api_integration:my-app-/my/route/1-GET", Destination: "aws:api_resource:my-app-/my/route/1"},
					{Source: "aws:api_integration:my-app-/my/route/1-GET", Destination: "aws:lambda_function:lambda"},
					{Source: "aws:api_integration:my-app-/my/route/1-GET", Destination: "aws:rest_api:api"},
					{Source: "aws:api_integration:my-app-/my/route/1-POST", Destination: "aws:api_method:my-app-/my/route/1-POST"},
					{Source: "aws:api_integration:my-app-/my/route/1-POST", Destination: "aws:api_resource:my-app-/my/route/1"},
					{Source: "aws:api_integration:my-app-/my/route/1-POST", Destination: "aws:lambda_function:lambda"},
					{Source: "aws:api_integration:my-app-/my/route/1-POST", Destination: "aws:rest_api:api"},
					{Source: "aws:api_integration:my-app-/my/route/2-POST", Destination: "aws:api_method:my-app-/my/route/2-POST"},
					{Source: "aws:api_integration:my-app-/my/route/2-POST", Destination: "aws:api_resource:my-app-/my/route/2"},
					{Source: "aws:api_integration:my-app-/my/route/2-POST", Destination: "aws:lambda_function:lambda"},
					{Source: "aws:api_integration:my-app-/my/route/2-POST", Destination: "aws:rest_api:api"},
					{Source: "aws:api_method:my-app-/-GET", Destination: "aws:rest_api:api"},
					{Source: "aws:api_method:my-app-/my/route/1-GET", Destination: "aws:api_resource:my-app-/my/route/1"},
					{Source: "aws:api_method:my-app-/my/route/1-GET", Destination: "aws:rest_api:api"},
					{Source: "aws:api_method:my-app-/my/route/1-POST", Destination: "aws:api_resource:my-app-/my/route/1"},
					{Source: "aws:api_method:my-app-/my/route/1-POST", Destination: "aws:rest_api:api"},
					{Source: "aws:api_method:my-app-/my/route/2-POST", Destination: "aws:api_resource:my-app-/my/route/2"},
					{Source: "aws:api_method:my-app-/my/route/2-POST", Destination: "aws:rest_api:api"},
					{Source: "aws:api_resource:my-app-/my", Destination: "aws:rest_api:api"},
					{Source: "aws:api_resource:my-app-/my/route", Destination: "aws:api_resource:my-app-/my"},
					{Source: "aws:api_resource:my-app-/my/route", Destination: "aws:rest_api:api"},
					{Source: "aws:api_resource:my-app-/my/route/1", Destination: "aws:api_resource:my-app-/my/route"},
					{Source: "aws:api_resource:my-app-/my/route/1", Destination: "aws:rest_api:api"},
					{Source: "aws:api_resource:my-app-/my/route/2", Destination: "aws:api_resource:my-app-/my/route"},
					{Source: "aws:api_resource:my-app-/my/route/2", Destination: "aws:rest_api:api"},
					{Source: "aws:lambda_permission:lambda_awsrest_apiapi", Destination: "aws:lambda_function:lambda"},
					{Source: "aws:lambda_permission:lambda_awsrest_apiapi", Destination: "aws:rest_api:api"},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			dag.AddDependencyWithData(tt.edge.Source, tt.edge.Destination, tt.edge.Properties.Data)
			kb, err := GetAwsKnowledgeBase()
			if !assert.NoError(err) {
				return
			}
			err = kb.ExpandEdge(&tt.edge, dag, "my-app")
			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)
		})
	}
}
