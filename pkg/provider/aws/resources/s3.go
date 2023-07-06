package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
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
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		ForceDestroy  bool
		IndexDocument string
	}

	S3Object struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Bucket        *S3Bucket
		Key           string
		FilePath      string
	}

	S3BucketPolicy struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Bucket        *S3Bucket
		Policy        *PolicyDocument
	}
)

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (bucket *S3Bucket) BaseConstructsRef() core.BaseConstructSet {
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

func (bucket *S3Bucket) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

type S3BucketCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
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

	bucket.ConstructsRef = params.Refs.Clone()
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

type S3ObjectCreateParams struct {
	AppName  string
	Refs     core.BaseConstructSet
	Name     string
	Key      string
	FilePath string
}

func (object *S3Object) Create(dag *core.ResourceGraph, params S3ObjectCreateParams) error {
	object.Name = objectSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	if dag.GetResource(object.Id()) != nil {
		return fmt.Errorf(`S3Object with name %s already exists`, object.Name)
	}
	object.ConstructsRef = params.Refs.Clone()
	object.Key = params.Key
	object.FilePath = params.FilePath
	dag.AddResource(object)
	return nil
}

func (object *S3Object) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if object.Bucket == nil {
		buckets := core.GetDownstreamResourcesOfType[*S3Bucket](dag, object)
		if len(buckets) != 1 {
			return fmt.Errorf("S3Object %s has %d bucket dependencies", object.Name, len(buckets))
		}
		object.Bucket = buckets[0]
	}
	dag.AddDependenciesReflect(object)
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (object *S3Object) BaseConstructsRef() core.BaseConstructSet {
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

func (bucket *S3Object) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

type S3BucketPolicyCreateParams struct {
	Name    string
	AppName string
	Refs    core.BaseConstructSet
}

func (policy *S3BucketPolicy) Create(dag *core.ResourceGraph, params S3BucketPolicyCreateParams) error {
	policy.Name = objectSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	policy.ConstructsRef = params.Refs.Clone()
	if dag.GetResource(policy.Id()) != nil {
		return errors.Errorf(`a bucket policy named "%s" already exists (internal error)`, policy.Id().String())
	}
	dag.AddResource(policy)
	return nil
}

func (policy *S3BucketPolicy) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if policy.Bucket == nil {
		buckets := core.GetDownstreamResourcesOfType[*S3Bucket](dag, policy)
		if len(buckets) != 1 {
			return fmt.Errorf("S3Object %s has %d bucket dependencies", policy.Name, len(buckets))
		}
		policy.Bucket = buckets[0]
	}
	dag.AddDependenciesReflect(policy)
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (policy *S3BucketPolicy) BaseConstructsRef() core.BaseConstructSet {
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

func (policy *S3BucketPolicy) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
