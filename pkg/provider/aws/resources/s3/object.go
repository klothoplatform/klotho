package s3

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const S3_OBJECT_TYPE = "s3_object"

var objectSanitizer = aws.S3ObjectSanitizer

type (
	S3Object struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Bucket        *S3Bucket
		Key           string
		FilePath      string
	}
)

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
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) KlothoConstructRef() []core.AnnotationKey {
	return object.ConstructsRef
}

// ID returns the id of the cloud resource
func (object *S3Object) Id() string {
	return fmt.Sprintf("%s:%s:%s", object.Provider(), S3_OBJECT_TYPE, object.Name)
}
