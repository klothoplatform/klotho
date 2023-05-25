package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_CreateS3Origin(t *testing.T) {
	unit := &core.StaticUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}, "test")
	distro := NewCloudfrontDistribution("test", "1")

	assert := assert.New(t)
	dag := core.NewResourceGraph()
	CreateS3Origin(unit, bucket, distro, dag)

	want := coretesting.ResourcesExpectation{
		Nodes: []string{
			"aws:cloudfront_distribution:test-1",
			"aws:cloudfront_origin_access_identity:test-bucket-test",
			"aws:s3_bucket:test-bucket",
			"aws:s3_bucket_policy:test-bucket-test",
		},
		Deps: []coretesting.StringDep{
			{Source: "aws:cloudfront_distribution:test-1", Destination: "aws:cloudfront_origin_access_identity:test-bucket-test"},
			{Source: "aws:s3_bucket_policy:test-bucket-test", Destination: "aws:cloudfront_origin_access_identity:test-bucket-test"},
			{Source: "aws:s3_bucket_policy:test-bucket-test", Destination: "aws:s3_bucket:test-bucket"},
		},
	}
	want.Assert(t, dag)

	oai := dag.GetResource(core.ResourceId{Provider: "aws", Type: "cloudfront_origin_access_identity", Name: fmt.Sprintf("%s-%s", bucket.Name, unit.ID)})
	if !assert.NotNil(oai) {
		return
	}
	res := dag.GetResource(core.ResourceId{Provider: "aws", Type: "s3_bucket_policy", Name: fmt.Sprintf("%s-%s", bucket.Name, unit.ID)})
	if !assert.NotNil(res) {
		return
	}
	bucketPolicy, ok := res.(*S3BucketPolicy)
	if !assert.True(ok) {
		return
	}

	assert.Equal(bucketPolicy.Policy.Statement[0], StatementEntry{
		Effect: "Allow",
		Principal: &Principal{
			AWS: core.IaCValue{
				Resource: oai,
				Property: IAM_ARN_IAC_VALUE,
			},
		},
		Action: []string{"s3:GetObject"},
		Resource: []core.IaCValue{
			{
				Resource: bucket,
				Property: ALL_BUCKET_DIRECTORY_IAC_VALUE,
			},
		},
	})

	if !assert.Len(distro.Origins, 1) {
		return
	}
	s3Origin := distro.Origins[0]
	assert.Equal(s3Origin.DomainName, core.IaCValue{
		Resource: bucket,
		Property: BUCKET_REGIONAL_DOMAIN_NAME_IAC_VALUE,
	})
	assert.Equal(s3Origin.OriginId, unit.ID)
	assert.Equal(s3Origin.S3OriginConfig, S3OriginConfig{
		OriginAccessIdentity: core.IaCValue{
			Resource: oai,
			Property: CLOUDFRONT_ACCESS_IDENTITY_PATH_IAC_VALUE,
		},
	})

}

func Test_CreateCustomOrigin(t *testing.T) {
	gw := &core.Gateway{AnnotationKey: core.AnnotationKey{ID: "test"}}
	apiStage := NewApiStage(NewApiDeployment(NewRestApi("test", gw), nil, nil), "stage", nil)
	distro := NewCloudfrontDistribution("test", "1")

	assert := assert.New(t)
	CreateCustomOrigin(gw, apiStage, distro)

	assert.Len(distro.Origins, 1)
	customOrigin := distro.Origins[0]

	assert.Equal(customOrigin.CustomOriginConfig, CustomOriginConfig{
		HttpPort:             80,
		HttpsPort:            443,
		OriginProtocolPolicy: "https-only",
		OriginSslProtocols:   []string{"SSLv3", "TLSv1", "TLSv1.1", "TLSv1.2"},
	})
	assert.Equal(customOrigin.DomainName, core.IaCValue{
		Resource: apiStage,
		Property: STAGE_INVOKE_URL_IAC_VALUE,
	})
	assert.Equal(customOrigin.OriginPath, "/"+apiStage.StageName)
	assert.Equal(customOrigin.OriginId, gw.ID)
}
