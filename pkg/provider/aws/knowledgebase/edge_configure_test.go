package knowledgebase

import (
	"testing"

	dgraph "github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_ConfigureEdge(t *testing.T) {
	orm := &core.Orm{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	cases := []struct {
		name string
		edge []graph.Edge[core.Resource]
		want []core.Resource
	}{
		{
			name: "single rds lambda",
			edge: []graph.Edge[core.Resource]{
				{
					Source:      &resources.LambdaFunction{Name: "lambda", Subnets: []*resources.Subnet{{Name: "sub1"}}, Role: &resources.IamRole{}, EnvironmentVariables: make(resources.EnvironmentVariables)},
					Destination: &resources.RdsInstance{Name: "rds"},
					Properties: dgraph.EdgeProperties{
						Data: knowledgebase.EdgeData{
							AppName:              "my-app",
							Source:               &resources.LambdaFunction{Name: "lambda"},
							Destination:          &resources.RdsInstance{Name: "rds"},
							EnvironmentVariables: []core.EnvironmentVariable{core.GenerateOrmConnStringEnvVar(orm)},
						},
					},
				},
			},
			want: []core.Resource{&resources.LambdaFunction{
				Name:                 "lambda",
				Subnets:              []*resources.Subnet{{Name: "sub1"}},
				Role:                 &resources.IamRole{},
				EnvironmentVariables: resources.EnvironmentVariables{"TEST_PERSIST_ORM_CONNECTION": core.IaCValue{Resource: &resources.RdsInstance{Name: "rds"}, Property: string(core.CONNECTION_STRING)}},
			},
				&resources.RdsInstance{Name: "rds"},
			},
		},
		{
			name: "single rds proxy and lambda",
			edge: []graph.Edge[core.Resource]{
				{
					Source:      &resources.LambdaFunction{Name: "lambda", Subnets: []*resources.Subnet{{Name: "sub1"}}, Role: &resources.IamRole{}, EnvironmentVariables: make(resources.EnvironmentVariables)},
					Destination: &resources.RdsProxy{Name: "rds", Role: &resources.IamRole{Name: "ProxyRole"}, Auths: []*resources.ProxyAuth{{SecretArn: core.IaCValue{Resource: &resources.Secret{Name: "Secret"}}}}},
					Properties: dgraph.EdgeProperties{
						Data: knowledgebase.EdgeData{
							AppName:              "my-app",
							Source:               &resources.LambdaFunction{Name: "lambda"},
							Destination:          &resources.RdsInstance{Name: "rds"},
							EnvironmentVariables: []core.EnvironmentVariable{core.GenerateOrmConnStringEnvVar(orm)},
						},
					},
				},
				{
					Source:      &resources.RdsProxyTargetGroup{Name: "rds", RdsInstance: &resources.RdsInstance{Name: "instance", CredentialsPath: "rds"}},
					Destination: &resources.RdsProxy{Name: "rds", Role: &resources.IamRole{Name: "ProxyRole"}, Auths: []*resources.ProxyAuth{{SecretArn: core.IaCValue{Resource: &resources.Secret{Name: "Secret"}}}}},
					Properties: dgraph.EdgeProperties{
						Data: knowledgebase.EdgeData{
							AppName:              "my-app",
							Source:               &resources.LambdaFunction{Name: "lambda"},
							Destination:          &resources.RdsInstance{Name: "rds"},
							EnvironmentVariables: []core.EnvironmentVariable{core.GenerateOrmConnStringEnvVar(orm)},
						},
					},
				},
				{
					Source:      &resources.RdsProxy{Name: "rds"},
					Destination: &resources.Secret{Name: "Secret"},
				},
				{
					Source:      &resources.SecretVersion{Name: "sv"},
					Destination: &resources.Secret{Name: "Secret"},
				},
				{
					Source:      &resources.RdsProxy{Name: "rds"},
					Destination: &resources.Secret{Name: "Secret"},
				},
				{
					Source: &resources.RdsProxyTargetGroup{
						Name: "rds",
						RdsProxy: &resources.RdsProxy{
							Name:  "rds",
							Auths: []*resources.ProxyAuth{{SecretArn: core.IaCValue{Resource: &resources.Secret{Name: "Secret"}}}},
						},
					},
					Destination: &resources.RdsInstance{Name: "instance", CredentialsFile: &core.FileRef{FPath: "rds"}, CredentialsPath: "rds"},
					Properties: dgraph.EdgeProperties{
						Data: knowledgebase.EdgeData{
							AppName:              "my-app",
							Source:               &resources.LambdaFunction{Name: "lambda"},
							Destination:          &resources.RdsInstance{Name: "rds"},
							EnvironmentVariables: []core.EnvironmentVariable{core.GenerateOrmConnStringEnvVar(orm)},
						},
					},
				},
			},
			want: []core.Resource{
				&resources.LambdaFunction{
					Name:    "lambda",
					Subnets: []*resources.Subnet{{Name: "sub1"}},
					Role:    &resources.IamRole{},
					EnvironmentVariables: resources.EnvironmentVariables{"TEST_PERSIST_ORM_CONNECTION": core.IaCValue{
						Resource: &resources.RdsProxy{
							Name:  "rds",
							Role:  &resources.IamRole{Name: "ProxyRole"},
							Auths: []*resources.ProxyAuth{{SecretArn: core.IaCValue{Resource: &resources.Secret{Name: "Secret"}}}},
						},
						Property: string(core.CONNECTION_STRING),
					},
					},
				},
				&resources.RdsProxy{
					Name:  "rds",
					Role:  &resources.IamRole{Name: "ProxyRole"},
					Auths: []*resources.ProxyAuth{{SecretArn: core.IaCValue{Resource: &resources.Secret{Name: "Secret"}}}},
				},
				&resources.SecretVersion{Name: "sv", Path: "rds", Type: "string"},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			for _, edge := range tt.edge {
				dag.AddDependencyWithData(edge.Source, edge.Destination, edge.Properties.Data)
			}
			kb, err := GetAwsKnowledgeBase()
			if !assert.NoError(err) {
				return
			}

			err = kb.ConfigureFromEdgeData(dag)
			if !assert.NoError(err) {
				return
			}
			for _, res := range tt.want {
				assert.Equal(res, dag.GetResource(res.Id()))
			}
		})
	}
}
