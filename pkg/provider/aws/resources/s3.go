package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var objectSanitizer = aws.S3ObjectSanitizer
var bucketSanitizer = aws.S3BucketSanitizer

const (
	S3_BUCKET_TYPE        = "s3_bucket"
	S3_OBJECT_TYPE        = "s3_object"
	S3_BUCKET_POLICY_TYPE = "s3_bucket_policy"

	ALL_BUCKET_DIRECTORY_IAC_VALUE        = "all_bucket_directory"
	BUCKET_REGIONAL_DOMAIN_NAME_IAC_VALUE = "bucket_regional_domain_name"
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

	S3BucketPolicy struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Bucket        *S3Bucket
		Policy        *PolicyDocument
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) KlothoConstructRef() []core.AnnotationKey {
	return bucket.ConstructsRef
}

// Id returns the id of the cloud resource
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) KlothoConstructRef() []core.AnnotationKey {
	return object.ConstructsRef
}

// Id returns the id of the cloud resource
func (object *S3Object) Id() string {
	return fmt.Sprintf("%s:%s:%s", object.Provider(), S3_OBJECT_TYPE, object.Name)
}

func NewBucketPolicy(policyName string, bucket *S3Bucket, policy *PolicyDocument) *S3BucketPolicy {
	return &S3BucketPolicy{
		Name:          objectSanitizer.Apply(fmt.Sprintf("%s-%s", bucket.Name, policyName)),
		ConstructsRef: bucket.KlothoConstructRef(),
		Policy:        policy,
		Bucket:        bucket,
	}
}

// Provider returns name of the provider the resource is correlated to
func (policy *S3BucketPolicy) Provider() string {
	return AWS_PROVIDER
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *S3BucketPolicy) KlothoConstructRef() []core.AnnotationKey {
	return policy.ConstructsRef
}

// Id returns the id of the cloud resource
func (policy *S3BucketPolicy) Id() string {
	return fmt.Sprintf("%s:%s:%s", policy.Provider(), S3_BUCKET_POLICY_TYPE, policy.Name)
}
