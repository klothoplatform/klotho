package resources

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	API_GATEWAY_REST_TYPE                           = "rest_api"
	API_GATEWAY_RESOURCE_TYPE                       = "api_resource"
	API_GATEWAY_METHOD_TYPE                         = "api_method"
	API_GATEWAY_INTEGRATION_TYPE                    = "api_integration"
	VPC_LINK_TYPE                                   = "vpc_link"
	API_GATEWAY_DEPLOYMENT_TYPE                     = "api_deployment"
	API_GATEWAY_STAGE_TYPE                          = "api_stage"
	API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE = "child_resources"
	LAMBDA_INTEGRATION_URI_IAC_VALUE                = "lambda_integration_uri"
	ALL_RESOURCES_ARN_IAC_VALUE                     = "all_resources_arn"
	STAGE_INVOKE_URL_IAC_VALUE                      = "stage_invoke_url"
)

var restApiSanitizer = aws.RestApiSanitizer
var apiResourceSanitizer = aws.ApiResourceSanitizer

type (
	RestApi struct {
		Name             string
		ConstructsRef    core.AnnotationKeySet
		BinaryMediaTypes []string
	}

	ApiResource struct {
		Name           string
		ConstructsRef  core.AnnotationKeySet
		RestApi        *RestApi
		PathPart       string
		ParentResource *ApiResource
	}

	ApiMethod struct {
		Name              string
		ConstructsRef     core.AnnotationKeySet
		RestApi           *RestApi
		Resource          *ApiResource
		HttpMethod        string
		RequestParameters map[string]bool
		Authorization     string
	}

	VpcLink struct {
		ConstructsRef core.AnnotationKeySet
		Target        core.Resource
	}

	ApiIntegration struct {
		Name                  string
		ConstructsRef         core.AnnotationKeySet
		RestApi               *RestApi
		Resource              *ApiResource
		Method                *ApiMethod
		RequestParameters     map[string]string
		IntegrationHttpMethod string
		Type                  string
		ConnectionType        string
		VpcLink               *VpcLink
		Uri                   core.IaCValue
		Route                 string
	}

	ApiDeployment struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		RestApi       *RestApi
		Triggers      map[string]string
	}

	ApiStage struct {
		Name          string
		ConstructsRef core.AnnotationKeySet
		StageName     string
		RestApi       *RestApi
		Deployment    *ApiDeployment
	}
)

type RestApiCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	Name    string
}

func (api *RestApi) Create(dag *core.ResourceGraph, params RestApiCreateParams) error {

	name := restApiSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	if params.AppName == "" {
		name = restApiSanitizer.Apply(params.Name)
	}
	api.Name = name
	api.ConstructsRef = params.Refs

	existingApi := dag.GetResourceByVertexId(api.Id().String())
	if existingApi != nil {
		graphApi := existingApi.(*RestApi)
		graphApi.ConstructsRef.AddAll(params.Refs)
	} else {
		dag.AddResource(api)
	}
	return nil
}

type RestApiConfigureParams struct {
	BinaryMediaTypes []string
}

func (api *RestApi) Configure(params RestApiConfigureParams) error {
	api.BinaryMediaTypes = []string{"application/octet-stream", "image/*"}
	if len(params.BinaryMediaTypes) > 0 {
		api.BinaryMediaTypes = params.BinaryMediaTypes
	}
	return nil
}

// convertPath will convert the path stored in our gateway construct into a path that is functionally the same within
// api gateway.
//
// Iff `wildcardsToGreedy` is true, this will turn "*}" into "+}". Otherwise, it will turn "*}" into "}".
//
// The path will be manipulated so that:
//   - any : characters will be removed and replaced the item with surrounding brackets, to signal this is a path
//     parameter
//   - any escaped / will turn into a single /
//   - any wildcard route will be propagated to the api gateway standard format
func convertPath(path string, wildcardsToGreedy bool) string {
	path = regexp.MustCompile(":([^/]+)").ReplaceAllString(path, "{$1}")
	greedyMarker := ""
	if wildcardsToGreedy {
		greedyMarker = "+"
	}
	path = regexp.MustCompile("[*]}").ReplaceAllString(path, greedyMarker+"}")
	path = regexp.MustCompile("//").ReplaceAllString(path, "/")
	return path
}

type ApiResourceCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	Path    string
	ApiName string
}

func (resource *ApiResource) Create(dag *core.ResourceGraph, params ApiResourceCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Path))
	resource.Name = name
	resource.ConstructsRef = params.Refs

	existingResource := dag.GetResourceByVertexId(resource.Id().String())
	if existingResource != nil {
		graphResource := existingResource.(*ApiResource)
		graphResource.ConstructsRef.AddAll(params.Refs)
		return nil
	} else {
		segments := strings.Split(params.Path, "/")
		resource.PathPart = convertPath(segments[len(segments)-1], true)
		subParams := map[string]any{
			"RestApi": RestApiCreateParams{
				Refs: params.Refs,
				Name: params.ApiName,
			},
		}
		// The root path is already created in api gw so we dont want to attempt to create an empty resource
		if len(segments) > 1 && segments[len(segments)-2] != "" {
			subParams["ParentResource"] = ApiResourceCreateParams{
				AppName: params.AppName,
				Path:    strings.Join(segments[:len(segments)-1], "/"),
				Refs:    params.Refs,
				ApiName: params.ApiName,
			}
		}
		err := dag.CreateDependencies(resource, subParams)
		if err != nil {
			return err
		}
	}
	return nil
}

type ApiIntegrationCreateParams struct {
	AppName    string
	Refs       core.AnnotationKeySet
	Path       string
	ApiName    string
	HttpMethod string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi is correlated to the constructs which required its creation.
func (integration *ApiIntegration) Create(dag *core.ResourceGraph, params ApiIntegrationCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s-%s", params.AppName, params.Path, params.HttpMethod))
	integration.Name = name
	integration.ConstructsRef = params.Refs

	existingResource := dag.GetResourceByVertexId(integration.Id().String())
	if existingResource != nil {
		graphResource := existingResource.(*ApiIntegration)
		graphResource.ConstructsRef.AddAll(params.Refs)
	} else {
		subParams := map[string]any{
			"RestApi": RestApiCreateParams{
				Refs: params.Refs,
				Name: params.ApiName,
			},
			"Method": params,
		}
		if params.Path != "" && params.Path != "/" {
			subParams["Resource"] = ApiResourceCreateParams{
				AppName: params.AppName,
				Refs:    params.Refs,
				Path:    params.Path,
				ApiName: params.ApiName,
			}
		}
		err := dag.CreateDependencies(integration, subParams)
		if err != nil {
			return err
		}
	}
	return nil
}

type ApiMethodCreateParams struct {
	AppName    string
	Refs       core.AnnotationKeySet
	Path       string
	ApiName    string
	HttpMethod string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi is correlated to the constructs which required its creation.
func (method *ApiMethod) Create(dag *core.ResourceGraph, params ApiMethodCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s-%s", params.AppName, params.Path, params.HttpMethod))
	method.Name = name
	method.ConstructsRef = params.Refs
	method.HttpMethod = params.HttpMethod

	existingResource := dag.GetResource(method.Id())
	if existingResource != nil {
		graphResource := existingResource.(*ApiMethod)
		graphResource.ConstructsRef.AddAll(params.Refs)
	} else {
		subParams := map[string]any{
			"RestApi": RestApiCreateParams{
				Refs: params.Refs,
				Name: params.ApiName,
			},
		}
		if params.Path != "" && params.Path != "/" {
			subParams["Resource"] = ApiResourceCreateParams{
				AppName: params.AppName,
				Refs:    params.Refs,
				Path:    params.Path,
				ApiName: params.ApiName,
			}
		}

		err := dag.CreateDependencies(method, subParams)
		if err != nil {
			return err
		}
	}
	return nil
}

type ApiMethodConfigureParams struct {
	Authorization string
}

func (method *ApiMethod) Configure(params ApiMethodConfigureParams) error {
	method.Authorization = "None"
	if params.Authorization != "" {
		method.Authorization = params.Authorization
	}
	return nil
}

type ApiDeploymentCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	Name    string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi is correlated to the constructs which required its creation.
func (deployment *ApiDeployment) Create(dag *core.ResourceGraph, params ApiDeploymentCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	deployment.Name = name
	deployment.ConstructsRef = params.Refs

	existingDeployment := dag.GetResourceByVertexId(deployment.Id().String())
	if existingDeployment != nil {
		graphDeployment := existingDeployment.(*ApiDeployment)
		graphDeployment.ConstructsRef.AddAll(params.Refs)
	} else {
		err := dag.CreateDependencies(deployment, map[string]any{
			"RestApi": params,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type ApiStageCreateParams struct {
	AppName string
	Refs    core.AnnotationKeySet
	Name    string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi is correlated to the constructs which required its creation.
func (stage *ApiStage) Create(dag *core.ResourceGraph, params ApiStageCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	stage.Name = name
	stage.ConstructsRef = params.Refs

	existingResource := dag.GetResourceByVertexId(stage.Id().String())
	if existingResource != nil {
		graphResource := existingResource.(*ApiStage)
		graphResource.ConstructsRef.AddAll(params.Refs)
		return nil
	} else {
		err := dag.CreateDependencies(stage, map[string]any{
			"RestApi":    params,
			"Deployment": params,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type ApiStageConfigureParams struct {
	StageName string
}

// Configure sets the intristic characteristics of a vpc based on parameters passed in
func (stage *ApiStage) Configure(params ApiStageConfigureParams) error {
	stage.StageName = "stage"
	if params.StageName != "" {
		stage.StageName = params.StageName
	}
	return nil
}

func NewRestApi(appName string, gw *core.Gateway) *RestApi {
	return &RestApi{
		Name:             restApiSanitizer.Apply(fmt.Sprintf("%s-%s", appName, gw.ID)),
		ConstructsRef:    core.AnnotationKeySetOf(gw.AnnotationKey),
		BinaryMediaTypes: []string{"application/octet-stream", "image/*"},
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (api *RestApi) KlothoConstructRef() core.AnnotationKeySet {
	return api.ConstructsRef
}

// Id returns the id of the cloud resource
func (api *RestApi) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_REST_TYPE,
		Name:     api.Name,
	}
}

func NewApiResource(currSegment string, api *RestApi, refs core.AnnotationKeySet, pathPart string, parentResource *ApiResource) *ApiResource {
	return &ApiResource{
		Name:           apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", api.Name, currSegment)),
		ConstructsRef:  refs,
		RestApi:        api,
		PathPart:       pathPart,
		ParentResource: parentResource,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (res *ApiResource) KlothoConstructRef() core.AnnotationKeySet {
	return res.ConstructsRef
}

// Id returns the id of the cloud resource
func (res *ApiResource) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_RESOURCE_TYPE,
		Name:     res.Name,
	}
}

func NewApiMethod(resource *ApiResource, api *RestApi, refs core.AnnotationKeySet, httpMethod string, requestParams map[string]bool) *ApiMethod {
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

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (method *ApiMethod) KlothoConstructRef() core.AnnotationKeySet {
	return method.ConstructsRef
}

// Id returns the id of the cloud resource
func (method *ApiMethod) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_METHOD_TYPE,
		Name:     method.Name,
	}
}

func NewVpcLink(resource core.Resource, refs core.AnnotationKeySet) *VpcLink {
	return &VpcLink{
		ConstructsRef: refs,
		Target:        resource,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (link *VpcLink) KlothoConstructRef() core.AnnotationKeySet {
	return link.ConstructsRef
}

// Id returns the id of the cloud resource
func (res *VpcLink) Id() core.ResourceId {
	name := "<no target>"
	if res.Target != nil {
		name = res.Target.Id().String()
	}
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     VPC_LINK_TYPE,
		Name:     name,
	}
}

func (res *VpcLink) Name() string {
	return res.Id().Name
}

func NewApiIntegration(method *ApiMethod, refs core.AnnotationKeySet, integrationMethod string, integrationType string, vpcLink *VpcLink, uri core.IaCValue, requestParameters map[string]string) *ApiIntegration {
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
		RequestParameters:     requestParameters,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (integration *ApiIntegration) KlothoConstructRef() core.AnnotationKeySet {
	return integration.ConstructsRef
}

// Id returns the id of the cloud resource
func (integration *ApiIntegration) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_INTEGRATION_TYPE,
		Name:     integration.Name,
	}
}

func NewApiDeployment(api *RestApi, refs core.AnnotationKeySet, triggers map[string]string) *ApiDeployment {
	return &ApiDeployment{
		Name:          api.Name,
		ConstructsRef: refs,
		RestApi:       api,
		Triggers:      triggers,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (deployment *ApiDeployment) KlothoConstructRef() core.AnnotationKeySet {
	return deployment.ConstructsRef
}

// Id returns the id of the cloud resource
func (deployment *ApiDeployment) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_DEPLOYMENT_TYPE,
		Name:     deployment.Name,
	}
}

func NewApiStage(deployment *ApiDeployment, stageName string, refs core.AnnotationKeySet) *ApiStage {
	return &ApiStage{
		Name:          fmt.Sprintf("%s-%s", deployment.Name, stageName),
		ConstructsRef: refs,
		Deployment:    deployment,
		RestApi:       deployment.RestApi,
		StageName:     stageName,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (stage *ApiStage) KlothoConstructRef() core.AnnotationKeySet {
	return stage.ConstructsRef
}

// Id returns the id of the cloud resource
func (stage *ApiStage) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_STAGE_TYPE,
		Name:     stage.Name,
	}
}
