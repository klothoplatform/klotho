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

func Test_GenerateFsResources(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.PersistCapability}}
	bucket := resources.NewS3Bucket(fs, "test")
	actions := []string{"s3:*"}
	policyResources := []core.IaCValue{
		{Resource: bucket, Property: core.ARN_IAC_VALUE},
		{Resource: bucket, Property: resources.ALL_BUCKET_DIRECTORY_IAC_VALUE},
	}
	policyDoc := resources.CreateAllowPolicyDocument(actions, policyResources)
	policy := resources.NewIamPolicy("test", fs.Id(), fs.Provenance(), policyDoc)
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
			name: "generate fs",
			want: testResult{
				nodes: []core.Resource{
					bucket,
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
					bucket, policy,
				},
				deps: []graph.Edge[core.Resource]{
					{Source: policy, Destination: bucket},
				},
				policy: policy.Policy.Statement[0],
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
				assert.Truef(found, "Expected to find dependency for %s -> %s", dep.Source.Id(), dep.Destination.Id())
			}
			if len(tt.want.policy.Action) != 0 {
				for _, statement := range aws.PolicyGenerator.GetUnitPolicies(eu.Id())[0].Policy.Statement {
					foundArnVal := false
					foundDirVal := false
					for _, val := range statement.Resource {
						assert.Equal(val.Resource.Id(), bucket.Id())
						if val.Property == core.ARN_IAC_VALUE {
							foundArnVal = true
						}
						if val.Property == resources.ALL_BUCKET_DIRECTORY_IAC_VALUE {
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
