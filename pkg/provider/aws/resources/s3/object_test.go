package s3

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewS3Object(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	object := NewS3Object(bucket, "test-name", "test-key", "test-path")
	assert.Equal(object.Name, "test-app-test-test-name")
	assert.Equal(object.ConstructsRef, []core.AnnotationKey{fs.AnnotationKey})
	assert.Equal(object.FilePath, "test-path")
	assert.Equal(object.Key, "test-key")
	assert.Equal(object.Bucket, bucket)
}

func Test_ObjectProvider(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	object := NewS3Object(bucket, "test-name", "test-key", "test-path")
	assert.Equal(object.Provider(), resources.AWS_PROVIDER)
}

func Test_ObjectId(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	object := NewS3Object(bucket, "test-name", "test-key", "test-path")
	assert.Equal(object.Id(), "aws:s3_object:test-app-test-test-name")
}

func Test_ObjectKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(fs, "test-app")
	object := NewS3Object(bucket, "test-name", "test-key", "test-path")
	assert.Equal(object.KlothoConstructRef(), []core.AnnotationKey{fs.Provenance()})
}
