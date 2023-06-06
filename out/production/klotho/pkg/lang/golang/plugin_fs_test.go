package golang

import (
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_queryFS(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    *persistResult
		wantErr bool
	}{
		{
			name: "simple file blob",
			source: `
import (
	"gocloud.dev/blob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err := blob.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "simple var file blob",
			source: `
import (
	"gocloud.dev/blob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err = blob.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "simple var file blob",
			source: `
import (
	"gocloud.dev/blob"
)
var bucket *blob.Bucket
var err error
/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err = blob.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "aliased file blob",
			source: `
		import (
			alias "gocloud.dev/blob"
		)
		/**
		* @klotho::persist {
		*	id = "test"
		* }
		*/
		bucket, err := alias.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "wrong import no match",
			source: `
		import (
			"gocloud.dev/blobby"
		)
		/**
		* @klotho::persist {
		*	id = "test"
		* }
		*/
		bucket, err := blobby.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			f, err := core.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			annot, ok := f.Annotations()[core.AnnotationKey{Capability: "persist", ID: "test"}]

			if !assert.True(ok) {
				return
			}
			result := queryFS(f, annot)
			if tt.want == nil {
				assert.Nil(result)
				return
			}
			assert.Equal(tt.want.varName, result.varName)
		})
	}
}

func Test_Transform(t *testing.T) {
	type testResult struct {
		resource core.Fs
		content  string
	}
	tests := []struct {
		name    string
		source  string
		want    testResult
		wantErr bool
	}{
		{
			name: "simple file blob",
			source: `package fs
import (
	"gocloud.dev/blob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err := blob.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))
`,
			want: testResult{
				resource: core.Fs{
					AnnotationKey: core.AnnotationKey{
						ID:         "test",
						Capability: annotation.PersistCapability,
					},
				},
				content: `package fs

import (
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/blob"
)

/**
* @klotho::persist {
*	id = "test"
* }
*/
var _ = fmt.Sprintf("file://%s", path)
bucket, err := blob.OpenBucket(context.Background(), "s3://" + os.Getenv("TEST_BUCKET_NAME") + "?region=" + os.Getenv("AWS_REGION"))
`,
			},
		},
		{
			name: "long var file blob",
			source: `package fs
import (
	"gocloud.dev/blob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err = blob.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))
`,
			want: testResult{
				resource: core.Fs{
					AnnotationKey: core.AnnotationKey{
						ID:         "test",
						Capability: annotation.PersistCapability,
					},
				},
				content: `package fs

import (
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/blob"
)

/**
* @klotho::persist {
*	id = "test"
* }
*/
var _ = fmt.Sprintf("file://%s", path)
var bucket, err = blob.OpenBucket(context.Background(), "s3://" + os.Getenv("TEST_BUCKET_NAME") + "?region=" + os.Getenv("AWS_REGION"))
`,
			},
		},
		{
			name: "var deckaration file blob",
			source: `package fs
import (
	"gocloud.dev/blob"
)
var bucket *blob.Bucket
var err error
/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err = blob.OpenBucket(context.Background(), fmt.Sprintf("file://%s", path))
`,
			want: testResult{
				resource: core.Fs{
					AnnotationKey: core.AnnotationKey{
						ID:         "test",
						Capability: annotation.PersistCapability,
					},
				},
				content: `package fs

import (
	_ "gocloud.dev/blob/s3blob"
	"gocloud.dev/blob"
)

var bucket *blob.Bucket
var err error
/**
* @klotho::persist {
*	id = "test"
* }
*/
var _ = fmt.Sprintf("file://%s", path)
bucket, err = blob.OpenBucket(context.Background(), "s3://" + os.Getenv("TEST_BUCKET_NAME") + "?region=" + os.Getenv("AWS_REGION"))
`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := PersistFsPlugin{runtime: NoopRuntime{}}
			unit := core.ExecutionUnit{}

			f, err := core.NewSourceFile("test.go", strings.NewReader(tt.source), Language)
			if !assert.NoError(err) {
				return
			}
			annot, ok := f.Annotations()[core.AnnotationKey{Capability: "persist", ID: "test"}]

			if !assert.True(ok) {
				return
			}
			queryResult := queryFS(f, annot)
			result, err := p.transformFS(f, annot, queryResult, &unit)
			if tt.wantErr {
				assert.Error(err)
				return
			} else if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want.resource.Provenance(), result.Provenance())
			assert.Equal(tt.want.content, string(f.Program()))
		})
	}
}
