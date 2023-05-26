package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"github.com/pkg/errors"
)

var cloudfrontDistributionSanitizer = aws.CloudfrontDistributionSanitizer

const (
	CLOUDFRONT_DISTRIBUTION_TYPE              = "cloudfront_distribution"
	ORIGIN_ACCESS_IDENTITY_TYPE               = "cloudfront_origin_access_identity"
	IAM_ARN_IAC_VALUE                         = "iam_arn"
	CLOUDFRONT_ACCESS_IDENTITY_PATH_IAC_VALUE = "cloudfront_access_identity_path"
)

type (
	CloudfrontDistribution struct {
		Name                         string
		ConstructsRef                core.AnnotationKeySet
		Origins                      []*CloudfrontOrigin
		CloudfrontDefaultCertificate bool
		Enabled                      bool
		DefaultCacheBehavior         *DefaultCacheBehavior
		Restrictions                 *Restrictions
		DefaultRootObject            string
	}

	DefaultCacheBehavior struct {
		AllowedMethods       []string
		CachedMethods        []string
		TargetOriginId       string
		ForwardedValues      ForwardedValues
		MinTtl               int
		DefaultTtl           int
		MaxTtl               int
		ViewerProtocolPolicy string
	}

	ForwardedValues struct {
		QueryString bool
		Cookies     Cookies
	}

	Cookies struct {
		Forward string
	}

	Restrictions struct {
		GeoRestriction GeoRestriction
	}

	GeoRestriction struct {
		RestrictionType string
	}

	CloudfrontOrigin struct {
		DomainName         core.IaCValue
		OriginId           string
		OriginPath         string
		S3OriginConfig     S3OriginConfig
		CustomOriginConfig CustomOriginConfig
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
		ConstructsRef core.AnnotationKeySet
		Comment       string
	}
)

type OriginAccessIdentityCreateParams struct {
	Name string
	Refs core.AnnotationKeySet
}

func (oai *OriginAccessIdentity) Create(dag *core.ResourceGraph, params OriginAccessIdentityCreateParams) error {
	oai.Name = params.Name
	oai.ConstructsRef = params.Refs
	if dag.GetResource(oai.Id()) != nil {
		return fmt.Errorf(`an Origin Access Identity with name "%s" already exists`, oai.Name)
	}

	// This is technically a config, but it's always just this value, so it's fine (and convenient) to inline it here

	dag.AddResource(oai)
	return nil
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

type CloudfrontDistributionCreateParams struct {
	CdnId   string
	AppName string
	Refs    core.AnnotationKeySet
}

func (distro *CloudfrontDistribution) Create(dag *core.ResourceGraph, params CloudfrontDistributionCreateParams) error {
	distro.Name = cloudfrontDistributionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.CdnId))
	distro.ConstructsRef = params.Refs

	if dag.GetResource(distro.Id()) != nil {
		return errors.Errorf(`duplicate Cloudfront distribution "%s" (internal error)`, distro.Id())
	}
	dag.AddResource(distro)

	// Some defaults. Someday these may move to a Configure (and be configurable), but not yet.
	distro.DefaultCacheBehavior = &DefaultCacheBehavior{
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
	}
	distro.Restrictions = &Restrictions{
		GeoRestriction: GeoRestriction{RestrictionType: "none"},
	}
	distro.CloudfrontDefaultCertificate = true
	return nil
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (distro *CloudfrontDistribution) KlothoConstructRef() core.AnnotationKeySet {
	return distro.ConstructsRef
}

// Id returns the id of the cloud resource
func (distro *CloudfrontDistribution) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     CLOUDFRONT_DISTRIBUTION_TYPE,
		Name:     distro.Name,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (oai *OriginAccessIdentity) KlothoConstructRef() core.AnnotationKeySet {
	return oai.ConstructsRef
}

// Id returns the id of the cloud resource
func (oai *OriginAccessIdentity) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ORIGIN_ACCESS_IDENTITY_TYPE,
		Name:     oai.Name,
	}
}
