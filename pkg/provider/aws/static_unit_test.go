package aws

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateStaticUnitResources(t *testing.T) {
	unit := &core.StaticUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	unit.AddSharedFile(&core.FileRef{FPath: "test/Shared"})
	unit.AddStaticFile(&core.FileRef{FPath: "test/Static"})
	cases := []struct {
		name          string
		indexDocument string
		cfg           config.Application
		want          coretesting.ResourcesExpectation
		wantErr       bool
	}{
		{
			name: "generate static unit with no index document",
			cfg: config.Application{
				AppName: "test-app",
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:test-app-test",
					"aws:s3_object:test-app-test-Shared",
					"aws:s3_object:test-app-test-Static",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:s3_object:test-app-test-Shared", Destination: "aws:s3_bucket:test-app-test"},
					{Source: "aws:s3_object:test-app-test-Static", Destination: "aws:s3_bucket:test-app-test"},
				},
			},
		},
		{
			name: "generate static unit with index document as website",
			cfg: config.Application{
				AppName: "test-app",
			},
			indexDocument: "index.html",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:test-app-test",
					"aws:s3_object:test-app-test-Shared",
					"aws:s3_object:test-app-test-Static",
				},
				Deps: []graph.Edge[string]{
					{Source: "aws:s3_object:test-app-test-Shared", Destination: "aws:s3_bucket:test-app-test"},
					{Source: "aws:s3_object:test-app-test-Static", Destination: "aws:s3_bucket:test-app-test"},
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
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			tiedResources, found := aws.GetResourcesDirectlyTiedToConstruct(unit)
			assert.True(found)
			mappedIds := []string{}
			for _, res := range tiedResources {
				mappedIds = append(mappedIds, res.Id())
			}
			assert.ElementsMatch(tt.want.Nodes, mappedIds)
			for _, res := range dag.ListResources() {
				if bucket, ok := res.(*resources.S3Bucket); ok {
					assert.Equal(bucket.IndexDocument, tt.indexDocument)

				}
			}
		})

	}
}
