package s3

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const S3_BUCKET_TYPE = "s3_bucket"

var sanitizer = aws.S3BucketSanitizer

type (
	S3Bucket struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		AccountId     *resources.AccountId
		ForceDestroy  bool
	}
)

func NewS3Bucket(fs *core.Fs, appName string, accountId *resources.AccountId) *S3Bucket {
	return &S3Bucket{
		Name:          sanitizer.Apply(fmt.Sprintf("%s-%s", appName, fs.ID)),
		ConstructsRef: []core.AnnotationKey{fs.Provenance()},
		AccountId:     accountId,
		ForceDestroy:  true,
	}
}

// Provider returns name of the provider the resource is correlated to
func (bucket *S3Bucket) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) KlothoConstructRef() []core.AnnotationKey {
	return bucket.ConstructsRef
}

// ID returns the id of the cloud resource
func (bucket *S3Bucket) Id() string {
	return fmt.Sprintf("%s:%s:%s", bucket.Provider(), S3_BUCKET_TYPE, bucket.Name)
}
