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
				assert.Equal(bucket.ConstructsRef, core.AnnotationKeySetOf(annotationKey))
			},
		},
		{
			Name: "two payloads buckets converge",
			Existing: []core.Resource{&S3Bucket{
				Name: "my-app-payloads",
				ConstructsRef: core.AnnotationKeySetOf(core.AnnotationKey{
					ID:         "some-other-eu",
					Capability: annotation.ExecutionUnitCapability}),
			}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:my-app-payloads",
				},
				Deps: nil,
			},
			Check: func(assert *assert.Assertions, bucket *S3Bucket) {
				assert.Equal("my-app-payloads", bucket.Name)
				assert.Equal(bucket.ConstructsRef,
					core.AnnotationKeySetOf(
						annotationKey,
						core.AnnotationKey{
							ID:         "some-other-eu",
							Capability: annotation.ExecutionUnitCapability},
					),
				)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = S3BucketCreateParams{
				AppName: "my-app",
				Refs:    core.AnnotationKeySetOf(annotationKey),
				Name:    "payloads",
			}
			tt.Run(t)
		})
	}
}

func Test_S3ObjectCreate(t *testing.T) {
	annotationKey := core.AnnotationKey{ID: "test", Capability: annotation.StaticUnitCapability}
	cases := []coretesting.CreateCase[S3ObjectCreateParams, *S3Object]{
		{
			Name: "s3 bucket missing",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_object:my-app-test-payloads",
					"aws:s3_bucket:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:s3_object:my-app-test-payloads", Destination: "aws:s3_bucket:my-app-test"},
				},
			},
			Check: func(assertions *assert.Assertions, object *S3Object) {
				// nothing extra
			},
			WantErr: false,
		},
		{
			Name: "s3 bucket alrady there",
			Existing: []core.Resource{
				&S3Bucket{Name: "my-app-test"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_object:my-app-test-payloads",
					"aws:s3_bucket:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:s3_object:my-app-test-payloads", Destination: "aws:s3_bucket:my-app-test"},
				},
			},
			Check: func(assertions *assert.Assertions, object *S3Object) {
				// nothing extra
			},
			WantErr: false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = S3ObjectCreateParams{
				AppName:  "my-app",
				Ref:      annotationKey,
				Name:     "payloads",
				Key:      "object-key",
				FilePath: "local/path.txt",
			}
			tt.Run(t)
		})
	}

}
