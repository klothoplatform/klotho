package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var CloudfrontKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.LoadBalancer]{},
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.S3Bucket]{
		Configure: func(distro *resources.CloudfrontDistribution, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			var errs multierr.Error
			for _, consRef := range distro.ConstructRefs {
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
			distro.DefaultRootObject = bucket.IndexDocument
			return errs.ErrOrNil()
		},
	},
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.ApiStage]{
		Configure: func(distro *resources.CloudfrontDistribution, stage *resources.ApiStage, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			var gwId string

			origin := &resources.CloudfrontOrigin{
				CustomOriginConfig: resources.CustomOriginConfig{
					HttpPort:             80,
					HttpsPort:            443,
					OriginProtocolPolicy: "https-only",
					OriginSslProtocols:   []string{"SSLv3", "TLSv1", "TLSv1.1", "TLSv1.2"},
				},
				DomainName: &resources.AwsResourceValue{
					ResourceVal: stage,
					PropertyVal: resources.STAGE_INVOKE_URL_IAC_VALUE,
				},
				OriginId:   gwId,
				OriginPath: &resources.AwsResourceValue{ResourceVal: stage, PropertyVal: resources.API_STAGE_PATH_VALUE},
			}
			fmt.Println(distro)
			fmt.Println(distro.Origins)
			distro.Origins = append(distro.Origins, origin)
			if distro.DefaultCacheBehavior == nil {
				distro.DefaultCacheBehavior = &resources.DefaultCacheBehavior{}
			}
			distro.DefaultCacheBehavior.TargetOriginId = origin.OriginId
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.OriginAccessIdentity]{},
)

type s3ToCloudfrontConnection struct {
	distro    *resources.CloudfrontDistribution
	bucket    *resources.S3Bucket
	dag       *core.ResourceGraph
	construct core.BaseConstruct
}

func (conn s3ToCloudfrontConnection) createOai() (*resources.OriginAccessIdentity, error) {
	oai, err := core.CreateResource[*resources.OriginAccessIdentity](conn.dag, resources.OriginAccessIdentityCreateParams{
		Name: fmt.Sprintf("%s-%s", conn.bucket.Name, conn.construct.Id().Name),
		Refs: core.BaseConstructSetOf(conn.construct),
	})
	if err != nil {
		return nil, err
	}
	conn.dag.AddDependency(conn.distro, oai)

	// This should be in an edge Configure, but it requires all three of the AOI, bucket, and distro -- so it's easier
	// to do it here, at create time when we already have all three.
	s3OriginConfig := resources.S3OriginConfig{
		OriginAccessIdentity: &resources.AwsResourceValue{
			ResourceVal: oai,
			PropertyVal: resources.CLOUDFRONT_ACCESS_IDENTITY_PATH_IAC_VALUE,
		},
	}
	origin := &resources.CloudfrontOrigin{
		S3OriginConfig: s3OriginConfig,
		DomainName: &resources.AwsResourceValue{
			ResourceVal: conn.bucket,
			PropertyVal: resources.BUCKET_REGIONAL_DOMAIN_NAME_IAC_VALUE,
		},
		OriginId: conn.construct.Id().Name,
	}
	conn.distro.Origins = append(conn.distro.Origins, origin)
	conn.distro.DefaultCacheBehavior.TargetOriginId = origin.OriginId
	return oai, err
}

func (conn s3ToCloudfrontConnection) attachPolicy(oai *resources.OriginAccessIdentity) error {
	policy, err := core.CreateResource[*resources.S3BucketPolicy](conn.dag, resources.S3BucketPolicyCreateParams{
		Name:    conn.construct.Id().Name,
		AppName: conn.bucket.Name,
		Refs:    core.BaseConstructSetOf(conn.construct),
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
					AWS: &resources.AwsResourceValue{
						ResourceVal: oai,
						PropertyVal: resources.IAM_ARN_IAC_VALUE,
					},
				},
				Action: []string{"s3:GetObject"},
				Resource: []*resources.AwsResourceValue{
					{
						ResourceVal: conn.bucket,
						PropertyVal: resources.ALL_BUCKET_DIRECTORY_IAC_VALUE,
					},
				},
			},
		},
	}
	return err
}
