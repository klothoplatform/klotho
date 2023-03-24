package s3

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewLambdaFunction(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test@#$#@$^%#$&wacjyksdjhgklSJDHGFAJHGT3O474oh43o"}}
	accountId := resources.NewAccountId()
	bucket := NewS3Bucket(fs, "test-app", accountId)
	assert.Equal(bucket.Name, "test-app-test-----------wacjyksdjhgkl-----------3-47")
	assert.Equal(bucket.ConstructsRef, []core.AnnotationKey{fs.AnnotationKey})
	assert.Equal(bucket.ForceDestroy, true)
	assert.Equal(bucket.AccountId, accountId)
}

func Test_Provider(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	accountId := resources.NewAccountId()
	bucket := NewS3Bucket(fs, "test-app", accountId)
	assert.Equal(bucket.Provider(), resources.AWS_PROVIDER)
}

func Test_Id(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	accountId := resources.NewAccountId()
	bucket := NewS3Bucket(fs, "test-app", accountId)
	assert.Equal(bucket.Id(), "aws:s3_bucket:test-app-test")
}

func Test_KlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	fs := &core.Fs{AnnotationKey: core.AnnotationKey{ID: "test"}}
	accountId := resources.NewAccountId()
	bucket := NewS3Bucket(fs, "test-app", accountId)
	assert.Equal(bucket.KlothoConstructRef(), []core.AnnotationKey{fs.Provenance()})
}
