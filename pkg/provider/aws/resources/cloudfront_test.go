package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_CreateS3Origin(t *testing.T) {

	unit := &core.StaticUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	bucket := NewS3Bucket(&core.Fs{AnnotationKey: core.AnnotationKey{ID: "bucket"}}, "test")
	distro := NewCloudfrontDistribution("test", "1")

	assert := assert.New(t)
	dag := core.NewResourceGraph()
	CreateS3Origin(unit, bucket, distro, dag)

	assert.NotNil(dag.GetDependency("aws:s3_bucket_policy:test-bucket-test", "aws:cloudfront_origin_access_identity:test-bucket-test"))
	assert.NotNil(dag.GetDependency("aws:s3_bucket_policy:test-bucket-test", "aws:s3_bucket:test-bucket"))
	assert.NotNil(dag.GetDependency("aws:cloudfront_distribution:test-1", "aws:cloudfront_origin_access_identity:test-bucket-test"))

	oai := dag.GetResource(fmt.Sprintf("aws:cloudfront_origin_access_identity:%s-%s", bucket.Name, unit.ID))
	if !assert.NotNil(oai) {
		return
	}
	res := dag.GetResource(fmt.Sprintf("aws:s3_bucket_policy:%s-%s", bucket.Name, unit.ID))
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
				Property: core.ARN_IAC_VALUE,
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

	assert.Len(distro.Origins, 1)
	s3Origin := distro.Origins[0]
	assert.Equal(s3Origin.DomainName, core.IaCValue{
		Resource: bucket,
		Property: BUCKET_REGIONAL_DOMAIN_NAME_IAC_VALUE,
	})
	assert.Equal(s3Origin.OriginId, unit.ID)
	assert.Equal(s3Origin.S3OriginConfig, S3OriginConfig{
		OriginAccessIdentity: core.IaCValue{
			Resource: oai,
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
