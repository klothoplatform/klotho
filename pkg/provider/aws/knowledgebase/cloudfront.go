package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var CloudfrontKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.S3Bucket]{
		Expand: func(distro *resources.CloudfrontDistribution, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			var errs multierr.Error
			for consRef := range distro.ConstructsRef {
				conn := s3ToCloudfrontConnection{
					distro:    distro,
					bucket:    bucket,
					dag:       dag,
					construct: consRef,
				}
				oai, err := conn.createOai()
				if err != nil {
					errs.Append(err)
					continue
				}
				err = conn.attachPolicy(oai)
				errs.Append(err)
			}
			return errs.ErrOrNil()
		},
		Configure: func(distro *resources.CloudfrontDistribution, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			distro.DefaultRootObject = bucket.IndexDocument
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.OriginAccessIdentity]{},
)

type s3ToCloudfrontConnection struct {
	distro    *resources.CloudfrontDistribution
	bucket    *resources.S3Bucket
	dag       *core.ResourceGraph
	construct core.AnnotationKey
}

func (conn s3ToCloudfrontConnection) createOai() (*resources.OriginAccessIdentity, error) {
	oai, err := core.CreateResource[*resources.OriginAccessIdentity](conn.dag, resources.OriginAccessIdentityCreateParams{
		Name: fmt.Sprintf("%s-%s", conn.bucket.Name, conn.construct.ID),
		Refs: core.AnnotationKeySetOf(conn.construct),
	})
	if err != nil {
		return nil, err
	}
	conn.dag.AddDependency(conn.distro, oai)

	// This should be in an edge Configure, but it requires all three of the AOI, bucket, and distro -- so it's easier
	// to do it here, at create time when we already have all three.
	s3OriginConfig := resources.S3OriginConfig{
		OriginAccessIdentity: core.IaCValue{
			Resource: oai,
			Property: resources.CLOUDFRONT_ACCESS_IDENTITY_PATH_IAC_VALUE,
		},
	}
	origin := &resources.CloudfrontOrigin{
		S3OriginConfig: s3OriginConfig,
		DomainName: core.IaCValue{
			Resource: conn.bucket,
			Property: resources.BUCKET_REGIONAL_DOMAIN_NAME_IAC_VALUE,
		},
		OriginId: conn.construct.ID,
	}
	conn.distro.Origins = append(conn.distro.Origins, origin)
	conn.distro.DefaultCacheBehavior.TargetOriginId = origin.OriginId
	return oai, err
}

func (conn s3ToCloudfrontConnection) attachPolicy(oai *resources.OriginAccessIdentity) error {
	policy, err := core.CreateResource[*resources.S3BucketPolicy](conn.dag, resources.S3BucketPolicyCreateParams{
		Name:       conn.construct.ID,
		BucketName: conn.bucket.Name,
		Refs:       core.AnnotationKeySetOf(conn.construct),
	})
	if err != nil {
		return err
	}
	conn.dag.AddDependency(policy, conn.bucket)
	conn.dag.AddDependency(policy, oai)
	policy.Policy = &resources.PolicyDocument{
		Version: resources.VERSION,
		Statement: []resources.StatementEntry{
			{
				Effect: "Allow",
				Principal: &resources.Principal{
					AWS: core.IaCValue{
						Resource: oai,
						Property: resources.IAM_ARN_IAC_VALUE,
					},
				},
				Action: []string{"s3:GetObject"},
				Resource: []core.IaCValue{
					{
						Resource: conn.bucket,
						Property: resources.ALL_BUCKET_DIRECTORY_IAC_VALUE,
					},
				},
			},
		},
	}
	return err
}
