package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

var cloudfrontDistributionSanitizer = aws.CloudfrontDistributionSanitizer

const (
	CLOUDFRONT_DISTRIBUTION_TYPE = "cloudfront_distribution"
	ORIGIN_ACCESS_IDENTITY_TYPE  = "cloudfront_origin_access_identity"
)

type (
	CloudfrontDistribution struct {
		Name                         string
		ConstructsRef                []core.AnnotationKey
		Origins                      []*CloudfrontOrigin `render:"document"`
		CloudfrontDefaultCertificate bool
		Enabled                      bool
		DefaultCacheBehavior         *DefaultCacheBehavior `render:"document"`
		Restrictions                 *Restrictions         `render:"document"`
		DefaultRootObject            string
	}

	DefaultCacheBehavior struct {
		AllowedMethods       []string
		CachedMethods        []string
		TargetOriginId       string
		ForwardedValues      ForwardedValues `render:"document"`
		MinTtl               int
		DefaultTtl           int
		MaxTtl               int
		ViewerProtocolPolicy string
	}

	ForwardedValues struct {
		QueryString bool
		Cookies     Cookies `render:"document"`
	}

	Cookies struct {
		Forward string
	}

	Restrictions struct {
		GeoRestriction GeoRestriction `render:"document"`
	}

	GeoRestriction struct {
		RestrictionType string
	}

	CloudfrontOrigin struct {
		DomainName         core.IaCValue
		OriginId           string
		OriginPath         string
		S3OriginConfig     S3OriginConfig     `render:"document"`
		CustomOriginConfig CustomOriginConfig `render:"document"`
	}

	S3OriginConfig struct {
		OriginAccessIdentity core.IaCValue
	}

	CustomOriginConfig struct {
		HttpPort             int
		HttpsPort            int
		OriginProtocolPolicy string
		OriginSslProtocols   []string
	}

	OriginAccessIdentity struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Comment       string
	}
)

// CreateS3Origin creates an origin for a static unit, given its bucket, and attaches it to the Cloudfront distribution passed in
func CreateS3Origin(unit *core.StaticUnit, bucket *S3Bucket, distribution *CloudfrontDistribution, dag *core.ResourceGraph) {

	oai := &OriginAccessIdentity{
		Name:          fmt.Sprintf("%s-%s", bucket.Name, unit.ID),
		ConstructsRef: []core.AnnotationKey{unit.AnnotationKey},
		Comment:       "this is needed to setup s3 polices and make s3 not public.",
	}

	policyDoc := &PolicyDocument{
		Version: VERSION,
		Statement: []StatementEntry{
			{
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
			},
		},
	}
	bucketPolicy := NewBucketPolicy(unit.ID, bucket, policyDoc)
	dag.AddDependency(bucketPolicy, oai)
	dag.AddDependency(distribution, oai)
	dag.AddDependency(bucketPolicy, bucket)
	s3OriginConfig := S3OriginConfig{
		OriginAccessIdentity: core.IaCValue{
			Resource: oai,
		},
	}
	origin := &CloudfrontOrigin{
		S3OriginConfig: s3OriginConfig,
		DomainName: core.IaCValue{
			Resource: bucket,
			Property: BUCKET_REGIONAL_DOMAIN_NAME_IAC_VALUE,
		},
		OriginId: unit.ID,
	}
	distribution.Origins = append(distribution.Origins, origin)
	distribution.DefaultCacheBehavior.TargetOriginId = origin.OriginId
}

// CreateCustomOrigin creates an origin for a gateway, given its api stage, and attaches it to the Cloudfront distribution passed in
func CreateCustomOrigin(gw *core.Gateway, apiStage *ApiStage, distribution *CloudfrontDistribution) {
	origin := &CloudfrontOrigin{
		CustomOriginConfig: CustomOriginConfig{
			HttpPort:             80,
			HttpsPort:            443,
			OriginProtocolPolicy: "https-only",
			OriginSslProtocols:   []string{"SSLv3", "TLSv1", "TLSv1.1", "TLSv1.2"},
		},
		DomainName: core.IaCValue{
			Resource: apiStage,
			Property: STAGE_INVOKE_URL_IAC_VALUE,
		},
		OriginId:   gw.ID,
		OriginPath: fmt.Sprintf("/%s", apiStage.StageName),
	}
	distribution.Origins = append(distribution.Origins, origin)
	distribution.DefaultCacheBehavior.TargetOriginId = origin.OriginId
}

func NewCloudfrontDistribution(appName string, cdnId string) *CloudfrontDistribution {
	return &CloudfrontDistribution{
		Name: cloudfrontDistributionSanitizer.Apply(fmt.Sprintf("%s-%s", appName, cdnId)),
		DefaultCacheBehavior: &DefaultCacheBehavior{
			AllowedMethods: []string{"DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"},
			CachedMethods:  []string{"HEAD", "GET"},
			ForwardedValues: ForwardedValues{
				QueryString: true,
				Cookies:     Cookies{Forward: "none"},
			},
			MinTtl:               0,
			DefaultTtl:           3600,
			MaxTtl:               86400,
			ViewerProtocolPolicy: "allow-all",
		},
		Restrictions: &Restrictions{
			GeoRestriction: GeoRestriction{RestrictionType: "none"},
		},
		CloudfrontDefaultCertificate: true,
	}
}

// Provider returns name of the provider the resource is correlated to
func (distro *CloudfrontDistribution) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (distro *CloudfrontDistribution) KlothoConstructRef() []core.AnnotationKey {
	return distro.ConstructsRef
}

// ID returns the id of the cloud resource
func (distro *CloudfrontDistribution) Id() string {
	return fmt.Sprintf("%s:%s:%s", distro.Provider(), CLOUDFRONT_DISTRIBUTION_TYPE, distro.Name)
}

// Provider returns name of the provider the resource is correlated to
func (oai *OriginAccessIdentity) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (oai *OriginAccessIdentity) KlothoConstructRef() []core.AnnotationKey {
	return oai.ConstructsRef
}

// ID returns the id of the cloud resource
func (oai *OriginAccessIdentity) Id() string {
	return fmt.Sprintf("%s:%s:%s", oai.Provider(), ORIGIN_ACCESS_IDENTITY_TYPE, oai.Name)
}
