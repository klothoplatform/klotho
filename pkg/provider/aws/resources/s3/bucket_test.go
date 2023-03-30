package s3

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewS3Bucket(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test@#$#@$^%#$&wacjyksdjhgklSJDHGFAJHGT3O474oh43o"}}
	bucket := NewS3Bucket(fs, "test-app")
	assert.Equal(bucket.Name, "test-app-test-----------wacjyksdjhgkl-----------3-47")
	assert.Equal(bucket.ConstructsRef, []core.AnnotationKey{fs.AnnotationKey})
	assert.Equal(bucket.ForceDestroy, true)
}

func Test_BucketProvider(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	assert.Equal(bucket.Provider(), resources.AWS_PROVIDER)
}

func Test_BucketId(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	assert.Equal(bucket.Id(), "aws:s3_bucket:test-app-test")
}

func Test_BucketKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	assert.Equal(bucket.KlothoConstructRef(), []core.AnnotationKey{fs.Provenance()})
}
