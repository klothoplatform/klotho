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

func (lambda *S3Bucket) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func (lambda *S3Object) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func (lambda *S3BucketPolicy) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func NewS3Bucket(fs core.Construct, appName string) *S3Bucket {
	return &S3Bucket{
		Name:          bucketSanitizer.Apply(fmt.Sprintf("%s-%s", appName, fs.Provenance().ID)),
		ConstructsRef: []core.AnnotationKey{fs.Provenance()},
		ForceDestroy:  true,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) KlothoConstructRef() []core.AnnotationKey {
	return bucket.ConstructsRef
}

// Id returns the id of the cloud resource
func (bucket *S3Bucket) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     S3_BUCKET_TYPE,
		Name:     bucket.Name,
	}
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) KlothoConstructRef() []core.AnnotationKey {
	return object.ConstructsRef
}

// Id returns the id of the cloud resource
func (object *S3Object) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     S3_OBJECT_TYPE,
		Name:     object.Name,
	}
}

func NewBucketPolicy(policyName string, bucket *S3Bucket, policy *PolicyDocument) *S3BucketPolicy {
	return &S3BucketPolicy{
		Name:          objectSanitizer.Apply(fmt.Sprintf("%s-%s", bucket.Name, policyName)),
		ConstructsRef: bucket.KlothoConstructRef(),
		Policy:        policy,
		Bucket:        bucket,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *S3BucketPolicy) KlothoConstructRef() []core.AnnotationKey {
	return policy.ConstructsRef
}

// Id returns the id of the cloud resource
func (policy *S3BucketPolicy) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     S3_BUCKET_POLICY_TYPE,
		Name:     policy.Name,
	}
}
