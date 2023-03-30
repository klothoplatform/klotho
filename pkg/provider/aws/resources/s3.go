package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var objectSanitizer = aws.S3ObjectSanitizer
var bucketSanitizer = aws.S3BucketSanitizer

const (
	S3_BUCKET_TYPE = "s3_bucket"
	S3_OBJECT_TYPE = "s3_object"
)

type (
	S3Bucket struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		ForceDestroy  bool
		IndexDocument string
	}

	S3Object struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Bucket        *S3Bucket
		Key           string
		FilePath      string
	}
)

func NewS3Bucket(fs core.Construct, appName string) *S3Bucket {
	return &S3Bucket{
		Name:          bucketSanitizer.Apply(fmt.Sprintf("%s-%s", appName, fs.Provenance().ID)),
		ConstructsRef: []core.AnnotationKey{fs.Provenance()},
		ForceDestroy:  true,
	}
}

// Provider returns name of the provider the resource is correlated to
func (bucket *S3Bucket) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) KlothoConstructRef() []core.AnnotationKey {
	return bucket.ConstructsRef
}

// ID returns the id of the cloud resource
func (bucket *S3Bucket) Id() string {
	return fmt.Sprintf("%s:%s:%s", bucket.Provider(), S3_BUCKET_TYPE, bucket.Name)
}

func NewS3Object(bucket *S3Bucket, objectName string, key string, path string) *S3Object {
	return &S3Object{
		Name:          objectSanitizer.Apply(fmt.Sprintf("%s-%s", bucket.Name, objectName)),
		ConstructsRef: bucket.KlothoConstructRef(),
		Key:           key,
		FilePath:      path,
		Bucket:        bucket,
	}
}

// Provider returns name of the provider the resource is correlated to
func (object *S3Object) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) KlothoConstructRef() []core.AnnotationKey {
	return object.ConstructsRef
}

// ID returns the id of the cloud resource
func (object *S3Object) Id() string {
	return fmt.Sprintf("%s:%s:%s", object.Provider(), S3_OBJECT_TYPE, object.Name)
}
