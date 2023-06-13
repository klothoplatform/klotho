package knowledgebase

import (
	"fmt"
	"sort"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

var CloudfrontKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.S3Bucket]{
		Expand: func(distro *resources.CloudfrontDistribution, bucket *resources.S3Bucket, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			var errs multierr.Error
			for _, consRef := range distro.ConstructsRef {
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
	knowledgebase.EdgeBuilder[*resources.CloudfrontDistribution, *resources.ApiStage]{
		Configure: func(distro *resources.CloudfrontDistribution, stage *resources.ApiStage, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			var gwId string
			switch len(stage.ConstructsRef) {
			case 0:
				return errors.Errorf(`couldn't determine the id of the construct that created API stage "%s"`, stage.Id())
			case 1:
				for cons := range stage.ConstructsRef {
					gwId = cons.Name
				}
			default:
				var ids []string
				for cons := range stage.ConstructsRef {
					ids = append(ids, cons.Name)
				}
				sort.Strings(ids)
				return errors.Errorf(
					`couldn't determine the id of the construct that created API stage "%s": expected just one construct, but found %v`,
					stage.Id(),
					ids)

			}
			origin := &resources.CloudfrontOrigin{
				CustomOriginConfig: resources.CustomOriginConfig{
					HttpPort:             80,
					HttpsPort:            443,
					OriginProtocolPolicy: "https-only",
					OriginSslProtocols:   []string{"SSLv3", "TLSv1", "TLSv1.1", "TLSv1.2"},
				},
				DomainName: core.IaCValue{
					Resource: stage,
					Property: resources.STAGE_INVOKE_URL_IAC_VALUE,
				},
				OriginId:   gwId,
				OriginPath: core.IaCValue{Resource: stage, Property: resources.API_STAGE_PATH_VALUE},
			}
			distro.Origins = append(distro.Origins, origin)
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
		OriginId: conn.construct.Id().Name,
	}
	conn.distro.Origins = append(conn.distro.Origins, origin)
	conn.distro.DefaultCacheBehavior.TargetOriginId = origin.OriginId
	return oai, err
}

func (conn s3ToCloudfrontConnection) attachPolicy(oai *resources.OriginAccessIdentity) error {
	policy, err := core.CreateResource[*resources.S3BucketPolicy](conn.dag, resources.S3BucketPolicyCreateParams{
		Name:       conn.construct.Id().Name,
		BucketName: conn.bucket.Name,
		Refs:       core.BaseConstructSetOf(conn.construct),
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
