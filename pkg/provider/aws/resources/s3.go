package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
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
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		ForceDestroy  bool
		IndexDocument string
	}

	S3Object struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Bucket        *S3Bucket
		Key           string
		FilePath      string
	}

	S3BucketPolicy struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Bucket        *S3Bucket
		Policy        *PolicyDocument
	}
)

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) BaseConstructRefs() construct.BaseConstructSet {
	return bucket.ConstructRefs
}

// Id returns the id of the cloud resource
func (bucket *S3Bucket) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     S3_BUCKET_TYPE,
		Name:     bucket.Name,
	}
}

func (bucket *S3Bucket) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

type S3BucketCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (bucket *S3Bucket) Create(dag *construct.ResourceGraph, params S3BucketCreateParams) error {
	bucket.Name = bucketSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))

	if existing := dag.GetResource(bucket.Id()); existing != nil {
		existingS3, ok := existing.(*S3Bucket)
		if !ok {
			return errors.Errorf(`found an existing element at %s, but it was not an S3Bucket`, bucket.Id().String())
		}
		// Multiple resources may create the same bucket (for example, this happens with our payload bucket and with
		// static unit S3Objects). If that happens, just append the refs and exit early; the rest would have been
		// idempotent.
		existingS3.ConstructRefs.AddAll(params.Refs)
		return nil
	}

	bucket.ConstructRefs = params.Refs.Clone()
	dag.AddResource(bucket)
	return nil
}

type S3ObjectCreateParams struct {
	AppName  string
	Refs     construct.BaseConstructSet
	Name     string
	Key      string
	FilePath string
}

func (object *S3Object) Create(dag *construct.ResourceGraph, params S3ObjectCreateParams) error {
	object.Name = objectSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	if dag.GetResource(object.Id()) != nil {
		return fmt.Errorf(`S3Object with name %s already exists`, object.Name)
	}
	object.ConstructRefs = params.Refs.Clone()
	object.Key = params.Key
	object.FilePath = params.FilePath
	dag.AddResource(object)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) BaseConstructRefs() construct.BaseConstructSet {
	return object.ConstructRefs
}

// Id returns the id of the cloud resource
func (object *S3Object) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     S3_OBJECT_TYPE,
		Name:     object.Name,
	}
}

func (bucket *S3Object) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

type S3BucketPolicyCreateParams struct {
	Name    string
	AppName string
	Refs    construct.BaseConstructSet
}

func (policy *S3BucketPolicy) Create(dag *construct.ResourceGraph, params S3BucketPolicyCreateParams) error {
	policy.Name = objectSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	policy.ConstructRefs = params.Refs.Clone()
	if dag.GetResource(policy.Id()) != nil {
		return errors.Errorf(`a bucket policy named "%s" already exists (internal error)`, policy.Id().String())
	}
	dag.AddResource(policy)
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *S3BucketPolicy) BaseConstructRefs() construct.BaseConstructSet {
	return policy.ConstructRefs
}

// Id returns the id of the cloud resource
func (policy *S3BucketPolicy) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     S3_BUCKET_POLICY_TYPE,
		Name:     policy.Name,
	}
}

func (policy *S3BucketPolicy) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
