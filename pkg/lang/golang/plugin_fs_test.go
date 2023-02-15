package golang

import (
	"strings"
	"testing"

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
	"gocloud.dev/blob/fileblob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err := fileblob.OpenBucket("myDir", nil)`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "simple var file blob",
			source: `
import (
	"gocloud.dev/blob/fileblob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err = fileblob.OpenBucket("myDir", nil)`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "simple var file blob",
			source: `
import (
	"gocloud.dev/blob/fileblob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err
bucket, err = fileblob.OpenBucket("myDir", nil)`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "aliased file blob",
			source: `
		import (
			alias "gocloud.dev/blob/fileblob"
		)
		/**
		* @klotho::persist {
		*	id = "test"
		* }
		*/
		bucket, err := alias.OpenBucket("myDir", nil)`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "non string as path still found in query",
			source: `
		import (
			alias "gocloud.dev/blob/fileblob"
		)
		/**
		* @klotho::persist {
		*	id = "test"
		* }
		*/
		bucket, err := alias.OpenBucket(myDir, nil)`,
			want: &persistResult{
				varName: "bucket",
			},
		},
		{
			name: "wrong import no match",
			source: `
		import (
			"gocloud.dev/blob/fileblobby"
		)
		/**
		* @klotho::persist {
		*	id = "test"
		* }
		*/
		bucket, err := fileblobby.OpenBucket(myDir, nil)`,
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
		resource core.Persist
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
	"gocloud.dev/blob/fileblob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err := fileblob.OpenBucket("myDir", nil)
`,
			want: testResult{
				resource: core.Persist{
					Kind: core.PersistFileKind,
					Name: "test",
				},
				content: `package fs

import (
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

/**
* @klotho::persist {
*	id = "test"
* }
*/
bucket, err := blob.OpenBucket(nil, "s3://" + os.Getenv("test_fs_bucket") + "?region=" + os.Getenv("AWS_REGION"))
`,
			},
		},
		{
			name: "long var file blob",
			source: `package fs
import (
	"gocloud.dev/blob/fileblob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err = fileblob.OpenBucket("myDir", nil)
`,
			want: testResult{
				resource: core.Persist{
					Kind: core.PersistFileKind,
					Name: "test",
				},
				content: `package fs

import (
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err = blob.OpenBucket(nil, "s3://" + os.Getenv("test_fs_bucket") + "?region=" + os.Getenv("AWS_REGION"))
`,
			},
		},
		{
			name: "var deckaration file blob",
			source: `package fs
import (
	"gocloud.dev/blob/fileblob"
)
/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err
bucket, err = fileblob.OpenBucket("myDir", nil)
`,
			want: testResult{
				resource: core.Persist{
					Kind: core.PersistFileKind,
					Name: "test",
				},
				content: `package fs

import (
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/s3blob"
)

/**
* @klotho::persist {
*	id = "test"
* }
*/
var bucket, err
bucket, err = blob.OpenBucket(nil, "s3://" + os.Getenv("test_fs_bucket") + "?region=" + os.Getenv("AWS_REGION"))
`,
			},
		},
		{
			name: "non string as path throws err",
			source: `package fs
		import (
			alias "gocloud.dev/blob/fileblob"
		)
		/**
		* @klotho::persist {
		*	id = "test"
		* }
		*/
		bucket, err := alias.OpenBucket(myDir, nil)`,
			wantErr: true,
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

			assert.Equal(tt.want.resource.Key(), result.Key())
			assert.Equal(tt.want.content, string(f.Program()))
		})
	}
}
