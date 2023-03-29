package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/iam"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/s3"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateFsResources(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	accountId := resources.NewAccountId()
	bucket := s3.NewS3Bucket(fs, "test", accountId)

	type testResult struct {
		nodes  []core.Resource
		deps   []graph.Edge[core.Resource]
		policy iam.StatementEntry
		err    bool
	}
	cases := []struct {
		name          string
		constructDeps []graph.Edge[core.Construct]
		want          testResult
	}{
		{
			name: "generate fs",
			want: testResult{
				nodes: []core.Resource{
					accountId, bucket,
				},
				deps: []graph.Edge[core.Resource]{
					{
						Source:      accountId,
						Destination: bucket,
					},
				},
			},
		},
		{
			name: "generate fs with upstream unit dep",
			constructDeps: []graph.Edge[core.Construct]{
				{
					Source:      eu,
					Destination: fs,
				},
			},
			want: testResult{
				nodes: []core.Resource{
					accountId, bucket,
				},
				deps: []graph.Edge[core.Resource]{
					{
						Source:      accountId,
						Destination: bucket,
					},
				},
				policy: iam.StatementEntry{
					Effect:   "Allow",
					Action:   []string{"s3:*"},
					Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE}},
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
				ConstructIdToResourceId: make(map[string]string),
				PolicyGenerator:         iam.NewPolicyGenerator(),
			}
			result := core.NewConstructGraph()

			for _, dep := range tt.constructDeps {
				result.AddConstruct(dep.Source)
				result.AddConstruct(dep.Destination)
				result.AddDependency(dep.Source.Id(), dep.Destination.Id())
			}

			dag := core.NewResourceGraph()
			err := aws.GenerateFsResources(fs, result, dag)
			if tt.want.err {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			for _, node := range tt.want.nodes {
				found := false
				for _, res := range dag.ListResources() {
					if res.Id() == node.Id() {
						found = true
					}
				}
				assert.True(found)
			}

			for _, dep := range tt.want.deps {
				found := false
				for _, res := range dag.ListDependencies() {
					if res.Source.Id() == dep.Source.Id() && res.Destination.Id() == dep.Destination.Id() {
						found = true
					}
				}
				assert.True(found)
			}
			if len(tt.want.policy.Action) != 0 {
				for _, statement := range aws.PolicyGenerator.GetUnitPolicies(eu.Id()).Statement {
					foundArnVal := false
					foundDirVal := false
					for _, val := range statement.Resource {
						assert.Equal(val.Resource.Id(), bucket.Id())
						if val.Property == core.ARN_IAC_VALUE {
							foundArnVal = true
						}
						if val.Property == core.ALL_BUCKET_DIRECTORY_IAC_VALUE {
							foundDirVal = true
						}
					}
					assert.True(foundArnVal)
					assert.True(foundDirVal)
					assert.ElementsMatch(statement.Action, tt.want.policy.Action)
					assert.Equal(statement.Effect, tt.want.policy.Effect)
				}
			}
		})

	}
}