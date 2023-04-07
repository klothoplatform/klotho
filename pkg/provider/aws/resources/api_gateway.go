package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	API_GATEWAY_REST_TYPE        = "rest_api"
	API_GATEWAY_RESOURCE_TYPE    = "api_resource"
	API_GATEWAY_METHOD_TYPE      = "api_method"
	API_GATEWAY_INTEGRATION_TYPE = "api_integration"
	VPC_LINK_TYPE                = "vpc_link"
	API_GATEWAY_DEPLOYMENT_TYPE  = "api_deployment"
	API_GATEWAY_STAGE_TYPE       = "api_stage"

	LAMBDA_INTEGRATION_URI_IAC_VALUE = "lambda_integration_uri"
	ALL_RESOURCES_ARN_IAC_VALUE      = "all_resources_arn"
	STAGE_INVOKE_URL_IAC_VALUE       = "stage_invoke_url"
)

var restApiSanitizer = aws.RestApiSanitizer
var apiResourceSanitizer = aws.ApiResourceSanitizer

type (
	RestApi struct {
		Name             string
		ConstructsRef    []core.AnnotationKey
		BinaryMediaTypes []string
	}

	ApiResource struct {
		Name           string
		ConstructsRef  []core.AnnotationKey
		RestApi        *RestApi
		PathPart       string
		ParentResource *ApiResource
	}

	ApiMethod struct {
		Name              string
		ConstructsRef     []core.AnnotationKey
		RestApi           *RestApi
		Resource          *ApiResource
		HttpMethod        string
		RequestParameters map[string]bool
		Authorization     string
	}

	VpcLink struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Target        core.Resource
	}

	ApiIntegration struct {
		Name                  string
		ConstructsRef         []core.AnnotationKey
		RestApi               *RestApi
		Resource              *ApiResource
		Method                *ApiMethod
		RequestParameters     map[string]string
		IntegrationHttpMethod string
		Type                  string
		ConnectionType        string
		VpcLink               *VpcLink
		Uri                   core.IaCValue
	}

	ApiDeployment struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		RestApi       *RestApi
		Triggers      map[string]string
	}

	ApiStage struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		StageName     string
		RestApi       *RestApi
		Deployment    *ApiDeployment
	}
)

func NewRestApi(appName string, gw *core.Gateway) *RestApi {
	return &RestApi{
		Name:             restApiSanitizer.Apply(fmt.Sprintf("%s-%s", appName, gw.ID)),
		ConstructsRef:    []core.AnnotationKey{gw.AnnotationKey},
		BinaryMediaTypes: []string{"application/octet-stream", "image/*"},
	}
}

// Provider returns name of the provider the resource is correlated to
func (api *RestApi) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (api *RestApi) KlothoConstructRef() []core.AnnotationKey {
	return api.ConstructsRef
}

// ID returns the id of the cloud resource
func (api *RestApi) Id() string {
	return fmt.Sprintf("%s:%s:%s", api.Provider(), API_GATEWAY_REST_TYPE, api.Name)
}

func NewApiResource(currSegment string, api *RestApi, refs []core.AnnotationKey, pathPart string, parentResource *ApiResource) *ApiResource {
	return &ApiResource{
		Name:           apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", api.Name, currSegment)),
		ConstructsRef:  refs,
		RestApi:        api,
		PathPart:       pathPart,
		ParentResource: parentResource,
	}
}

// Provider returns name of the provider the resource is correlated to
func (res *ApiResource) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (res *ApiResource) KlothoConstructRef() []core.AnnotationKey {
	return res.ConstructsRef
}

// ID returns the id of the cloud resource
func (res *ApiResource) Id() string {
	return fmt.Sprintf("%s:%s:%s", res.Provider(), API_GATEWAY_RESOURCE_TYPE, res.Name)
}

func NewApiMethod(resource *ApiResource, api *RestApi, refs []core.AnnotationKey, httpMethod string, requestParams map[string]bool) *ApiMethod {
	name := fmt.Sprintf("%s-%s", api.Name, httpMethod)
	if resource != nil {
		name = fmt.Sprintf("%s-%s", resource.Name, httpMethod)
	}
	return &ApiMethod{
		Name:              name,
		ConstructsRef:     refs,
		RestApi:           api,
		Resource:          resource,
		HttpMethod:        httpMethod,
		RequestParameters: requestParams,
		Authorization:     "None",
	}
}

// Provider returns name of the provider the resource is correlated to
func (method *ApiMethod) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (method *ApiMethod) KlothoConstructRef() []core.AnnotationKey {
	return method.ConstructsRef
}

// ID returns the id of the cloud resource
func (method *ApiMethod) Id() string {
	return fmt.Sprintf("%s:%s:%s", method.Provider(), API_GATEWAY_METHOD_TYPE, method.Name)
}

func NewVpcLink(resource core.Resource, refs []core.AnnotationKey) *VpcLink {
	return &VpcLink{
		Name:          resource.Id(),
		ConstructsRef: refs,
		Target:        resource,
	}
}

// Provider returns name of the provider the resource is correlated to
func (link *VpcLink) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (link *VpcLink) KlothoConstructRef() []core.AnnotationKey {
	return link.ConstructsRef
}

// ID returns the id of the cloud resource
func (link *VpcLink) Id() string {
	return fmt.Sprintf("%s:%s:%s", link.Provider(), VPC_LINK_TYPE, link.Name)
}

func NewApiIntegration(method *ApiMethod, refs []core.AnnotationKey, integrationMethod string, integrationType string, vpcLink *VpcLink, uri core.IaCValue) *ApiIntegration {
	return &ApiIntegration{
		Name:                  method.Name,
		ConstructsRef:         refs,
		RestApi:               method.RestApi,
		Resource:              method.Resource,
		Method:                method,
		IntegrationHttpMethod: integrationMethod,
		Type:                  integrationType,
		VpcLink:               vpcLink,
		Uri:                   uri,
	}
}

// Provider returns name of the provider the resource is correlated to
func (integration *ApiIntegration) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (integration *ApiIntegration) KlothoConstructRef() []core.AnnotationKey {
	return integration.ConstructsRef
}

// ID returns the id of the cloud resource
func (integration *ApiIntegration) Id() string {
	return fmt.Sprintf("%s:%s:%s", integration.Provider(), API_GATEWAY_INTEGRATION_TYPE, integration.Name)
}

func NewApiDeployment(api *RestApi, refs []core.AnnotationKey, triggers map[string]string) *ApiDeployment {
	return &ApiDeployment{
		Name:          api.Name,
		ConstructsRef: refs,
		RestApi:       api,
		Triggers:      triggers,
	}
}

// Provider returns name of the provider the resource is correlated to
func (deployment *ApiDeployment) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (deployment *ApiDeployment) KlothoConstructRef() []core.AnnotationKey {
	return deployment.ConstructsRef
}

// ID returns the id of the cloud resource
func (deployment *ApiDeployment) Id() string {
	return fmt.Sprintf("%s:%s:%s", deployment.Provider(), API_GATEWAY_DEPLOYMENT_TYPE, deployment.Name)
}

func NewApiStage(deployment *ApiDeployment, stageName string, refs []core.AnnotationKey) *ApiStage {
	return &ApiStage{
		Name:          fmt.Sprintf("%s-%s", deployment.Name, stageName),
		ConstructsRef: refs,
		Deployment:    deployment,
		RestApi:       deployment.RestApi,
		StageName:     stageName,
	}
}

// Provider returns name of the provider the resource is correlated to
func (stage *ApiStage) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (stage *ApiStage) KlothoConstructRef() []core.AnnotationKey {
	return stage.ConstructsRef
}

// ID returns the id of the cloud resource
func (stage *ApiStage) Id() string {
	return fmt.Sprintf("%s:%s:%s", stage.Provider(), API_GATEWAY_STAGE_TYPE, stage.Name)
}
