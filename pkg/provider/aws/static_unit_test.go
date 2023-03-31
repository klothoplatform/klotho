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

func Test_GenerateStaticUnitResources(t *testing.T) {
	unit := &core.StaticUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	unit.AddSharedFile(&core.FileRef{FPath: "test/Shared"})
	unit.AddStaticFile(&core.FileRef{FPath: "test/Static"})
	bucket := resources.NewS3Bucket(unit, "test-app")
	obj1 := resources.NewS3Object(bucket, "Shared", "test/Shared", "test/test/Shared")
	obj2 := resources.NewS3Object(bucket, "Static", "test/Static", "test/test/Static")

	type testResult struct {
		nodes []core.Resource
		deps  []graph.Edge[core.Resource]
		err   bool
	}
	cases := []struct {
		name          string
		indexDocument string
		cfg           config.Application
		want          testResult
	}{
		{
			name: "generate static unit with no index document",
			cfg: config.Application{
				AppName: "test-app",
			},
			want: testResult{
				nodes: []core.Resource{
					bucket, obj1, obj2,
				},
				deps: []graph.Edge[core.Resource]{
					{
						Source:      bucket,
						Destination: obj1,
					},
					{
						Source:      bucket,
						Destination: obj2,
					},
				},
			},
		},
		{
			name: "generate static unit with index document as website",
			cfg: config.Application{
				AppName: "test-app",
			},
			indexDocument: "index.html",
			want: testResult{
				nodes: []core.Resource{
					bucket, obj1, obj2,
				},
				deps: []graph.Edge[core.Resource]{
					{
						Source:      bucket,
						Destination: obj1,
					},
					{
						Source:      bucket,
						Destination: obj2,
					},
				},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			aws := AWS{
				Config: &tt.cfg,
			}
			dag := core.NewResourceGraph()
			unit.IndexDocument = tt.indexDocument

			err := aws.GenerateStaticUnitResources(unit, dag)
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
					if res.Id() == bucket.Id() {
						if b, ok := res.(*resources.S3Bucket); ok {
							assert.Equal(b.IndexDocument, tt.indexDocument)
						}
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
		})

	}
}
