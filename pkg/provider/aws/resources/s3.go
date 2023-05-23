package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"github.com/pkg/errors"
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

type S3BucketCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

func (bucket *S3Bucket) Create(dag *core.ResourceGraph, params S3BucketCreateParams) error {
	bucket.Name = bucketSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))

	if existing := dag.GetResource(bucket.Id()); existing != nil {
		existingS3, ok := existing.(*S3Bucket)
		if !ok {
			return errors.Errorf(`found an existing element at %s, but it was not an S3Bucket`, bucket.Id().String())
		}
		// Multiple resources may create the same bucket (today, this specifically happens with our payload bucket). If
		// that happens, just append the refs and exit early; the rest would have been idempotent.
		existingS3.ConstructsRef = collectionutil.FlattenUnique(existingS3.ConstructsRef, params.Refs)
		return nil
	}

	bucket.ConstructsRef = params.Refs
	dag.AddResource(bucket)
	return nil
}

type S3BucketConfigureParams struct {
	ForceDestroy  bool
	IndexDocument string
}

func (bucket *S3Bucket) Configure(params S3BucketConfigureParams) error {
	bucket.ForceDestroy = params.ForceDestroy
	bucket.IndexDocument = params.IndexDocument
	return nil
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
