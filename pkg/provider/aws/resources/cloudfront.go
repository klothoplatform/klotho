package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
	"github.com/pkg/errors"
)

var cloudfrontDistributionSanitizer = aws.CloudfrontDistributionSanitizer

const (
	CLOUDFRONT_DISTRIBUTION_TYPE              = "cloudfront_distribution"
	ORIGIN_ACCESS_IDENTITY_TYPE               = "cloudfront_origin_access_identity"
	IAM_ARN_IAC_VALUE                         = "iam_arn"
	API_STAGE_PATH_VALUE                      = "api_stage_name"
	CLOUDFRONT_ACCESS_IDENTITY_PATH_IAC_VALUE = "cloudfront_access_identity_path"
)

type (
	CloudfrontDistribution struct {
		Name                         string
		ConstructRefs                construct.BaseConstructSet `yaml:"-"`
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
		DomainName         construct.IaCValue
		OriginId           string
		OriginPath         construct.IaCValue
		S3OriginConfig     S3OriginConfig
		CustomOriginConfig CustomOriginConfig
	}

	S3OriginConfig struct {
		OriginAccessIdentity construct.IaCValue
	}

	CustomOriginConfig struct {
		HttpPort             int
		HttpsPort            int
		OriginProtocolPolicy string
		OriginSslProtocols   []string
	}

	OriginAccessIdentity struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Comment       string
	}
)

type OriginAccessIdentityCreateParams struct {
	Name string
	Refs construct.BaseConstructSet
}

func (oai *OriginAccessIdentity) Create(dag *construct.ResourceGraph, params OriginAccessIdentityCreateParams) error {
	oai.Name = params.Name
	oai.ConstructRefs = params.Refs.Clone()
	if dag.GetResource(oai.Id()) != nil {
		return fmt.Errorf(`an Origin Access Identity with name "%s" already exists`, oai.Name)
	}
	dag.AddResource(oai)
	return nil
}

type CloudfrontDistributionCreateParams struct {
	CdnId   string
	AppName string
	Refs    construct.BaseConstructSet
}

func (distro *CloudfrontDistribution) Create(dag *construct.ResourceGraph, params CloudfrontDistributionCreateParams) error {
	distro.Name = cloudfrontDistributionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.CdnId))
	distro.ConstructRefs = params.Refs.Clone()

	if dag.GetResource(distro.Id()) != nil {
		return errors.Errorf(`duplicate Cloudfront distribution "%s" (internal error)`, distro.Id())
	}
	dag.AddResource(distro)

	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (distro *CloudfrontDistribution) BaseConstructRefs() construct.BaseConstructSet {
	return distro.ConstructRefs
}

// Id returns the id of the cloud resource
func (distro *CloudfrontDistribution) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     CLOUDFRONT_DISTRIBUTION_TYPE,
		Name:     distro.Name,
	}
}
func (distro *CloudfrontDistribution) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (oai *OriginAccessIdentity) BaseConstructRefs() construct.BaseConstructSet {
	return oai.ConstructRefs
}

// Id returns the id of the cloud resource
func (oai *OriginAccessIdentity) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ORIGIN_ACCESS_IDENTITY_TYPE,
		Name:     oai.Name,
	}
}

func (oai *OriginAccessIdentity) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   false,
		RequiresNoDownstream: false,
	}
}
