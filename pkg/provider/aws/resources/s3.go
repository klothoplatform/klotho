package resources

import (
	"fmt"

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
		ConstructsRef core.AnnotationKeySet
		ForceDestroy  bool
		IndexDocument string
	}

	S3Object struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Bucket        *S3Bucket
		Key           string
		FilePath      string
	}

	S3BucketPolicy struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		Bucket        *S3Bucket
		Policy        *PolicyDocument
	}
)

func NewS3Bucket(fs core.Construct, appName string) *S3Bucket {
	return &S3Bucket{
		Name:          bucketSanitizer.Apply(fmt.Sprintf("%s-%s", appName, fs.Provenance().ID)),
		ConstructsRef: core.AnnotationKeySetOf(fs.Provenance()),
		ForceDestroy:  true,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) KlothoConstructRef() core.AnnotationKeySet {
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
	Refs    core.AnnotationKeySet
	Name    string
}

func (bucket *S3Bucket) Create(dag *core.ResourceGraph, params S3BucketCreateParams) error {
	bucket.Name = bucketSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))

	if existing := dag.GetResource(bucket.Id()); existing != nil {
		existingS3, ok := existing.(*S3Bucket)
		if !ok {
			return errors.Errorf(`found an existing element at %s, but it was not an S3Bucket`, bucket.Id().String())
		}
		// Multiple resources may create the same bucket (for example, this happens with our payload bucket and with
		// static unit S3Objects). If that happens, just append the refs and exit early; the rest would have been
		// idempotent.
		existingS3.ConstructsRef.AddAll(params.Refs)
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

//
//func NewS3Object(bucket *S3Bucket, objectName string, key string, path string) *S3Object {
//	return &S3Object{
//		Name:          objectSanitizer.Apply(fmt.Sprintf("%s-%s", bucket.Name, objectName)),
//		ConstructsRef: bucket.KlothoConstructRef(),
//		Key:           key,
//		FilePath:      path,
//		Bucket:        bucket,
//	}
//}

type S3ObjectCreateParams struct {
	AppName  string
	Refs     core.AnnotationKeySet
	UnitName string
	Name     string
	Key      string
	FilePath string
}

func (object *S3Object) Create(dag *core.ResourceGraph, params S3ObjectCreateParams) error {
	bucket, err := core.CreateResource[*S3Bucket](dag, S3BucketCreateParams{
		AppName: params.AppName,
		Refs:    params.Refs,
		Name:    params.UnitName,
	})
	if err != nil {
		return nil
	}

	object.Name = objectSanitizer.Apply(fmt.Sprintf("%s-%s", bucket.Name, params.Name))
	object.Bucket = bucket
	if dag.GetResource(object.Id()) != nil {
		return fmt.Errorf(`S3Object with name %s already exists`, object.Name)
	}
	object.ConstructsRef = params.Refs
	object.Key = params.Key
	object.FilePath = params.FilePath
	dag.AddDependency(object, bucket)
	return nil
}

type S3ObjectConfigureParams struct{}

func (object *S3Object) Configure(params S3ObjectConfigureParams) error {
	// nothing
	return nil
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) KlothoConstructRef() core.AnnotationKeySet {
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
func (policy *S3BucketPolicy) KlothoConstructRef() core.AnnotationKeySet {
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
