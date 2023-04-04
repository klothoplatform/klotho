package aws

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

// (The filename says this is a secret test! So shh, don't tell anyone about it!)

func TestGenerateSecretsResources(t *testing.T) {
	type testResult struct {
		resourceIds map[string]struct{}
		deps        []graph.Edge[string]
		policies    func(secretResolver func(string) *resources.SecretVersion) []resources.StatementEntry
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
					fmt.Sprintf(`aws:secret:%s-%s-secret1`, AppName, secretsConstructId):                              {},
					fmt.Sprintf(`aws:secret_version:%s-%s-secret1`, AppName, secretsConstructId):                      {},
					fmt.Sprintf(`aws:secret:%s-%s-secret2`, AppName, secretsConstructId):                              {},
					fmt.Sprintf(`aws:secret_version:%s-%s-secret2`, AppName, secretsConstructId):                      {},
					fmt.Sprintf(`aws:iam_policy:%s-%s_%s`, AppName, annotation.PersistCapability, secretsConstructId): {},
				},
				deps: []graph.Edge[string]{
					{
						Source:      fmt.Sprintf(`aws:iam_policy:%s-%s_%s`, AppName, annotation.PersistCapability, secretsConstructId),
						Destination: fmt.Sprintf(`aws:secret_version:%s-%s-secret1`, AppName, secretsConstructId),
					},
					{
						Source:      fmt.Sprintf(`aws:iam_policy:%s-%s_%s`, AppName, annotation.PersistCapability, secretsConstructId),
						Destination: fmt.Sprintf(`aws:secret_version:%s-%s-secret2`, AppName, secretsConstructId),
					},
					{
						Source:      fmt.Sprintf(`aws:secret:%s-%s-secret1`, AppName, secretsConstructId),
						Destination: fmt.Sprintf(`aws:secret_version:%s-%s-secret1`, AppName, secretsConstructId),
					},
					{
						Source:      fmt.Sprintf(`aws:secret:%s-%s-secret2`, AppName, secretsConstructId),
						Destination: fmt.Sprintf(`aws:secret_version:%s-%s-secret2`, AppName, secretsConstructId),
					},
				},
				policies: func(secretResolver func(string) *resources.SecretVersion) []resources.StatementEntry {
					secret1 := secretResolver(fmt.Sprintf(`aws:secret_version:%s-%s-secret1`, AppName, secretsConstructId))
					secret2 := secretResolver(fmt.Sprintf(`aws:secret_version:%s-%s-secret2`, AppName, secretsConstructId))
					return []resources.StatementEntry{
						{
							Effect: "Allow",
							Action: []string{`secretsmanager:DescribeSecret`, `secretsmanager:GetSecretValue`},
							Resource: []core.IaCValue{
								{
									Resource: secret1,
									Property: core.ARN_IAC_VALUE,
								},
							},
						},
						{
							Effect: "Allow",
							Action: []string{`secretsmanager:DescribeSecret`, `secretsmanager:GetSecretValue`},
							Resource: []core.IaCValue{
								{
									Resource: secret2,
									Property: core.ARN_IAC_VALUE,
								},
							},
						},
					}
				},
			},
		},
		{
			name:     "no secrets",
			execUnit: execUnit,
			want: testResult{
				resourceIds: map[string]struct{}{},
				deps:        nil,
				policies: func(_ func(string) *resources.SecretVersion) []resources.StatementEntry {
					return nil
				},
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
			secretsConstruct := &core.Secrets{
				AnnotationKey: core.AnnotationKey{ID: secretsConstructId, Capability: annotation.PersistCapability},
				Secrets:       tt.secrets,
			}

			constructGraph := core.NewConstructGraph()
			constructGraph.AddConstruct(secretsConstruct)
			if tt.execUnit != nil {
				constructGraph.AddConstruct(tt.execUnit)
				constructGraph.AddDependency(execUnit.Id(), secretsConstruct.Id())
			}

			dag := core.NewResourceGraph()
			err := aws.GenerateSecretsResources(secretsConstruct, constructGraph, dag)
			if !assert.NoError(err) {
				return
			}

			actualResourceIds := graph.VertexIds(dag.ListResources())
			assert.Equal(tt.want.resourceIds, actualResourceIds)

			graph.SortEdgeIds(tt.want.deps)
			actual := graph.SortEdgeIds(graph.EdgeIds(dag.ListDependencies()))
			assert.Equal(tt.want.deps, actual)

			wantPolicies := tt.want.policies(func(secretId string) *resources.SecretVersion {
				resource := dag.GetResource(secretId)
				assert.NotNil(resource)
				if secret, foundSecret := resource.(*resources.SecretVersion); foundSecret {
					return secret
				} else {
					assert.Failf(`found a resource with id="%s", but it wasn't a Secret: %v`, secretId, resource)
				}
				return nil
			})
			var actualPolicies []resources.StatementEntry
			if policies := aws.PolicyGenerator.GetUnitPolicies(execUnit.Id()); policies != nil {
				actualPolicies = policies[0].Policy.Statement
			}
			assert.Equal(wantPolicies, actualPolicies)
		})
	}
}
