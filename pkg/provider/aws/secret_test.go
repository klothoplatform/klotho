package aws

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
	"testing"
)

// (The filename says this is a secret test! So shh, don't tell anyone about it!)

func TestGenerateSecretsResources(t *testing.T) {
	type testResult struct {
		resourceIds map[string]struct{}
		deps        []graph.Edge[string]
		//policy resources.StatementEntry
	}
	const AppName = "AppName"
	const secretsConstructId = "MySecrets"

	execUnit := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "TestUnit", Capability: annotation.ExecutionUnitCapability}}

	cases := []struct {
		name     string
		secrets  []string
		execUnit *core.ExecutionUnit
		want     testResult
	}{
		{
			name:     "two secrets",
			secrets:  []string{`secret1`, `secret2`},
			execUnit: execUnit,
			want: testResult{
				resourceIds: map[string]struct{}{
					fmt.Sprintf(`aws:secret:%s-%s-secret1`, AppName, secretsConstructId):         {},
					fmt.Sprintf(`aws:secret_version:%s-%s-secret1`, AppName, secretsConstructId): {},
					fmt.Sprintf(`aws:secret:%s-%s-secret2`, AppName, secretsConstructId):         {},
					fmt.Sprintf(`aws:secret_version:%s-%s-secret2`, AppName, secretsConstructId): {},
				},
				deps: []graph.Edge[string]{
					{
						Source:      fmt.Sprintf(`aws:secret:%s-%s-secret1`, AppName, secretsConstructId),
						Destination: fmt.Sprintf(`aws:secret_version:%s-%s-secret1`, AppName, secretsConstructId),
					},
					{
						Source:      fmt.Sprintf(`aws:secret:%s-%s-secret2`, AppName, secretsConstructId),
						Destination: fmt.Sprintf(`aws:secret_version:%s-%s-secret2`, AppName, secretsConstructId),
					},
				},
			},
		},
		{
			name:     "no secrets",
			execUnit: execUnit,
			want: testResult{
				resourceIds: map[string]struct{}{},
				deps:        nil,
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: &config.Application{
					AppName: AppName,
				},
				PolicyGenerator: resources.NewPolicyGenerator(),
			}
			secretsRes := &core.Secrets{
				AnnotationKey: core.AnnotationKey{ID: secretsConstructId, Capability: annotation.PersistCapability},
				Secrets:       tt.secrets,
			}

			constructGraph := core.NewConstructGraph()
			constructGraph.AddConstruct(secretsRes)
			if tt.execUnit != nil {
				constructGraph.AddConstruct(tt.execUnit)
				constructGraph.AddDependency(secretsRes.Id(), execUnit.Id())
			}

			dag := core.NewResourceGraph()
			err := aws.GenerateSecretsResources(secretsRes, constructGraph, dag)
			if !assert.NoError(err) {
				return
			}

			actualResourceIds := graph.VertexIds(dag.ListResources())
			assert.Equal(tt.want.resourceIds, actualResourceIds)

			graph.SortEdgeIds(tt.want.deps)
			actual := graph.SortEdgeIds(graph.EdgeIds(dag.ListDependencies()))
			assert.Equal(tt.want.deps, actual)
		})
	}

}
