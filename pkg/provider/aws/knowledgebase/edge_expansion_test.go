package knowledgebase

import (
	"fmt"
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
					"aws:api_integration:api_integration_api_lambda",
					"aws:lambda_function:lambda",
					"aws:rest_api:api",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:api_integration:api_integration_api_lambda", Destination: "aws:lambda_function:lambda"},
					{Source: "aws:api_integration:api_integration_api_lambda", Destination: "aws:rest_api:api"},
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
			err = kb.ExpandEdge(&tt.edge, dag)
			if !assert.NoError(err) {
				return
			}
			fmt.Println(coretesting.ResoucesFromDAG(dag).GoString())
			tt.want.Assert(t, dag)
		})
	}
}
