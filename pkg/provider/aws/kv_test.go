package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateKVResources(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	kv := &core.Kv{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	table := resources.NewDynamodbTable(kv, "KV_test", []resources.DynamodbTableAttribute{
		{Name: "pk", Type: "S"},
		{Name: "sk", Type: "S"},
	})
	table.HashKey = "pk"
	table.RangeKey = "sk"
	table.BillingMode = resources.PAY_PER_REQUEST

	type testResult struct {
		nodes  []core.Resource
		deps   []graph.Edge[core.Resource]
		policy resources.StatementEntry
		err    bool
	}
	cases := []struct {
		name          string
		constructDeps []graph.Edge[core.Construct]
		want          testResult
	}{
		{
			name: "generate kv",
			want: testResult{
				nodes: []core.Resource{
					table,
				},
			},
		},
		{
			name: "generate kv with upstream unit dep",
			constructDeps: []graph.Edge[core.Construct]{
				{
					Source:      eu,
					Destination: kv,
				},
			},
			want: testResult{
				nodes: []core.Resource{
					table,
				},
				policy: resources.StatementEntry{
					Effect: "Allow",
					Action: []string{"dynamodb:*"},
					Resource: []core.IaCValue{
						{Resource: table, Property: core.ARN_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
					},
				},
			},
		},
		{
			name: "generate multiple kvs with upstream unit deps",
			constructDeps: []graph.Edge[core.Construct]{
				{
					Source:      eu,
					Destination: kv,
				},
				{
					Source:      eu,
					Destination: &core.Kv{AnnotationKey: core.AnnotationKey{ID: "second", Capability: annotation.PersistCapability}},
				},
			},
			want: testResult{
				nodes: []core.Resource{
					table,
				},
				policy: resources.StatementEntry{
					Effect: "Allow",
					Action: []string{"dynamodb:*"},
					Resource: []core.IaCValue{
						{Resource: table, Property: core.ARN_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
						{Resource: table, Property: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
					},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: &config.Application{
					AppName: "test",
				},
				PolicyGenerator: resources.NewPolicyGenerator(),
			}
			result := core.NewConstructGraph()
			dag := core.NewResourceGraph()

			err := aws.GenerateKvResources(kv, result, dag)
			if tt.want.err {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			for _, dep := range tt.constructDeps {
				result.AddConstruct(dep.Source)
				result.AddConstruct(dep.Destination)
				result.AddDependency(dep.Source.Id(), dep.Destination.Id())

				if dep, ok := dep.Destination.(*core.Kv); ok {
					err := aws.GenerateKvResources(dep, result, dag)
					if tt.want.err {
						assert.Error(err)
						return
					}
					if !assert.NoError(err) {
						return
					}
				}
			}

			for _, node := range tt.want.nodes {
				found := false
				for _, res := range dag.ListResources() {
					if res.Id() == node.Id() {
						found = true
					}
				}
				assert.Truef(found, "resource with id '%s' not found in resource graph", node.Id())
			}

			for _, dep := range tt.want.deps {
				found := false
				for _, res := range dag.ListDependencies() {
					if res.Source.Id() == dep.Source.Id() && res.Destination.Id() == dep.Destination.Id() {
						found = true
					}
				}
				assert.Truef(found, "dependency [%s -> %s] not found resource graph edges", dep.Source.Id(), dep.Destination.Id())
			}
			if len(tt.want.policy.Action) != 0 {
				statements := aws.PolicyGenerator.GetUnitPolicies(eu.Id()).Statement
				assert.Equal(len(tt.want.policy.Action), len(statements))
				for _, statement := range statements {
					for _, val := range statement.Resource {
						assert.Equal(val.Resource.Id(), table.Id())
					}
					assert.ElementsMatch(statement.Action, tt.want.policy.Action)
					assert.Equal(statement.Effect, tt.want.policy.Effect)
				}
			}
		})

	}
}
