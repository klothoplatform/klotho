package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_S3BucketCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[S3BucketCreateParams, *S3Bucket]{
		{
			Name: "nil bucket",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:my-app-bucket",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, bucket *S3Bucket) {
				assert.Equal(bucket.Name, "my-app-bucket")
				assert.Equal(bucket.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing bucket",
			Existing: &S3Bucket{Name: "my-app-bucket", ConstructsRef: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket:my-app-bucket",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, bucket *S3Bucket) {
				assert.Equal(bucket.Name, "my-app-bucket")
				assert.Equal(bucket.ConstructsRef, core.BaseConstructSetOf(eu, eu2))
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = S3BucketCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "bucket",
			}
			tt.Run(t)
		})
	}
}

func Test_S3ObjectCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[S3ObjectCreateParams, *S3Object]{
		{
			Name: "nil object",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_object:my-app-object",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, bucket *S3Object) {
				assert.Equal(bucket.Name, "my-app-object")
				assert.Equal(bucket.ConstructsRef, core.BaseConstructSetOf(eu))
				assert.Equal(bucket.Key, "key")
				assert.Equal(bucket.FilePath, "filepath")
			},
		},
		{
			Name:     "existing object",
			Existing: &S3Object{Name: "my-app-object", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = S3ObjectCreateParams{
				AppName:  "my-app",
				Refs:     core.BaseConstructSetOf(eu),
				Name:     "object",
				Key:      "key",
				FilePath: "filepath",
			}
			tt.Run(t)
		})
	}
}

func Test_S3BucketObjectMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*S3Object]{
		{
			Name:     "only object",
			Resource: &S3Object{Name: "secretv"},
			AppName:  "my-app",
			WantErr:  true,
		},
		{
			Name:     "sv with downstream secret",
			Resource: &S3Object{Name: "secretv"},
			AppName:  "my-app",
			Existing: []core.Resource{&S3Bucket{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:s3_object:secretv", Destination: "aws:s3_bucket:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_object:secretv",
					"aws:s3_bucket:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:s3_object:secretv", Destination: "aws:s3_bucket:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, sv *S3Object) {
				assert.Equal(sv.Bucket.Name, "test-down")
			},
		},
		{
			Name:     "sv with secret set",
			Resource: &S3Object{Name: "secretv", Bucket: &S3Bucket{Name: "s"}},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_object:secretv",
					"aws:s3_bucket:s",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:s3_object:secretv", Destination: "aws:s3_bucket:s"},
				},
			},
			Check: func(assert *assert.Assertions, sv *S3Object) {
				assert.Equal(sv.Bucket.Name, "s")
			},
		},
		{
			Name:     "multiple bucekts error",
			Resource: &S3Object{Name: "my_app"},
			AppName:  "my-app",
			Existing: []core.Resource{&S3Bucket{Name: "test-down"}, &S3Bucket{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:s3_object:my_app", Destination: "aws:s3_bucket:test-down"},
				{Source: "aws:s3_object:my_app", Destination: "aws:s3_bucket:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}

func Test_S3BucketPolicyCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[S3BucketPolicyCreateParams, *S3BucketPolicy]{
		{
			Name: "nil policy",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket_policy:my-app-policy",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, bucket *S3BucketPolicy) {
				assert.Equal(bucket.Name, "my-app-policy")
				assert.Equal(bucket.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing policy",
			Existing: &S3BucketPolicy{Name: "my-app-policy", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = S3BucketPolicyCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "policy",
			}
			tt.Run(t)
		})
	}
}

func Test_S3BucketPolicyMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*S3BucketPolicy]{
		{
			Name:     "only policy",
			Resource: &S3BucketPolicy{Name: "secretv"},
			AppName:  "my-app",
			WantErr:  true,
		},
		{
			Name:     "sv with downstream secret",
			Resource: &S3BucketPolicy{Name: "secretv"},
			AppName:  "my-app",
			Existing: []core.Resource{&S3Bucket{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:s3_bucket_policy:secretv", Destination: "aws:s3_bucket:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket_policy:secretv",
					"aws:s3_bucket:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:s3_bucket_policy:secretv", Destination: "aws:s3_bucket:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, sv *S3BucketPolicy) {
				assert.Equal(sv.Bucket.Name, "test-down")
			},
		},
		{
			Name:     "sv with secret set",
			Resource: &S3BucketPolicy{Name: "secretv", Bucket: &S3Bucket{Name: "s"}},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:s3_bucket_policy:secretv",
					"aws:s3_bucket:s",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:s3_bucket_policy:secretv", Destination: "aws:s3_bucket:s"},
				},
			},
			Check: func(assert *assert.Assertions, sv *S3BucketPolicy) {
				assert.Equal(sv.Bucket.Name, "s")
			},
		},
		{
			Name:     "multiple bucekts error",
			Resource: &S3BucketPolicy{Name: "my_app"},
			AppName:  "my-app",
			Existing: []core.Resource{&S3Bucket{Name: "test-down"}, &S3Bucket{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:s3_bucket_policy:my_app", Destination: "aws:s3_bucket:test-down"},
				{Source: "aws:s3_bucket_policy:my_app", Destination: "aws:s3_bucket:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
