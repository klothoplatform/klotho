package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_S3BucketCreate(t *testing.T) {
	annotationKey := core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}
	cases := []coretesting.CreateCase[S3BucketCreateParams, *S3Bucket]{
		{
			Name: "single payloads bucket",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:my-app-payloads",
				},
				Deps: nil,
			},
			Check: func(assert *assert.Assertions, bucket *S3Bucket) {
				assert.Equal("my-app-payloads", bucket.Name)
				assert.ElementsMatch(bucket.ConstructsRef, []core.AnnotationKey{annotationKey})
			},
		},
		{
			Name: "two payloads buckets converge",
			Existing: &S3Bucket{
				Name: "my-app-payloads",
				ConstructsRef: []core.AnnotationKey{core.AnnotationKey{
					ID:         "some-other-eu",
					Capability: annotation.ExecutionUnitCapability}},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:my-app-payloads",
				},
				Deps: nil,
			},
			Check: func(assert *assert.Assertions, bucket *S3Bucket) {
				assert.Equal("my-app-payloads", bucket.Name)
				assert.ElementsMatch(bucket.ConstructsRef,
					[]core.AnnotationKey{
						annotationKey,
						core.AnnotationKey{
							ID:         "some-other-eu",
							Capability: annotation.ExecutionUnitCapability},
					},
				)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = S3BucketCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{annotationKey},
				Name:    "payloads",
			}
			tt.Run(t)
		})
	}
}
