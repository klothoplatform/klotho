package resources

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
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
		ConstructRefs    construct.BaseConstructSet `yaml:"-"`
		BinaryMediaTypes []string
	}

	ApiResource struct {
		Name           string
		ConstructRefs  construct.BaseConstructSet `yaml:"-"`
		RestApi        *RestApi
		PathPart       string
		ParentResource *ApiResource
	}

	ApiMethod struct {
		Name              string
		ConstructRefs     construct.BaseConstructSet `yaml:"-"`
		RestApi           *RestApi
		Resource          *ApiResource
		HttpMethod        string
		RequestParameters map[string]bool
		Authorization     string
	}

	VpcLink struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Target        construct.ResourceId
	}

	ApiIntegration struct {
		Name                  string
		ConstructRefs         construct.BaseConstructSet `yaml:"-"`
		RestApi               *RestApi
		Resource              *ApiResource
		Method                *ApiMethod
		RequestParameters     map[string]string
		IntegrationHttpMethod string
		Type                  string
		ConnectionType        string
		VpcLink               *VpcLink
		Uri                   construct.IaCValue
		Route                 string
	}

	ApiDeployment struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		RestApi       *RestApi
		Triggers      map[string]string
	}

	ApiStage struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		StageName     string
		RestApi       *RestApi
		Deployment    *ApiDeployment
	}
)

type RestApiCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (api *RestApi) Create(dag *construct.ResourceGraph, params RestApiCreateParams) error {

	name := restApiSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	if params.AppName == "" {
		name = restApiSanitizer.Apply(params.Name)
	}
	api.Name = name
	api.ConstructRefs = params.Refs.Clone()

	existingApi := dag.GetResource(api.Id())
	if existingApi != nil {
		graphApi := existingApi.(*RestApi)
		graphApi.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(api)
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
	Refs    construct.BaseConstructSet
	Path    string
	ApiName string
}

func (resource *ApiResource) Create(dag *construct.ResourceGraph, params ApiResourceCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Path))
	resource.Name = name
	resource.ConstructRefs = params.Refs.Clone()

	existingResource := dag.GetResource(resource.Id())
	if existingResource != nil {
		graphResource := existingResource.(*ApiResource)
		graphResource.ConstructRefs.AddAll(params.Refs)
		return nil
	} else {
		segments := strings.Split(params.Path, "/")
		resource.PathPart = convertPath(segments[len(segments)-1], true)
		// The root path is already created in api gw so we dont want to attempt to create an empty resource
		if len(segments) > 1 && segments[len(segments)-2] != "" {
			parentResource, err := construct.CreateResource[*ApiResource](dag, ApiResourceCreateParams{
				AppName: params.AppName,
				Path:    strings.Join(segments[:len(segments)-1], "/"),
				Refs:    params.Refs,
				ApiName: params.ApiName,
			})
			if err != nil {
				return err
			}
			resource.ParentResource = parentResource
			dag.AddDependency(parentResource, resource)
		}
	}
	return nil
}

type ApiIntegrationCreateParams struct {
	AppName    string
	Refs       construct.BaseConstructSet
	Path       string
	ApiName    string
	HttpMethod string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi
// is correlated to the constructs which required its creation.
func (integration *ApiIntegration) Create(dag *construct.ResourceGraph, params ApiIntegrationCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s-%s", params.AppName, params.Path, params.HttpMethod))
	integration.Name = name
	integration.ConstructRefs = params.Refs.Clone()
	integration.Route = convertPath(params.Path, false)

	existingResource, found := construct.GetResource[*ApiIntegration](dag, integration.Id())
	if found {
		existingResource.ConstructRefs.AddAll(params.Refs)
		return nil
	} else {
		dag.AddResource(integration)
	}
	return nil
}

type ApiMethodCreateParams struct {
	AppName    string
	Refs       construct.BaseConstructSet
	Path       string
	ApiName    string
	HttpMethod string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi
// is correlated to the constructs which required its creation.
func (method *ApiMethod) Create(dag *construct.ResourceGraph, params ApiMethodCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s-%s", params.AppName, params.Path, params.HttpMethod))
	method.Name = name
	method.ConstructRefs = params.Refs.Clone()
	method.HttpMethod = params.HttpMethod

	existingResource := dag.GetResource(method.Id())
	if existingResource != nil {
		graphResource := existingResource.(*ApiMethod)
		graphResource.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(method)
		// The root path is already created in api gw so we dont want to attempt to create an empty resource
		if params.Path != "" && params.Path != "/" {
			parentResource, err := construct.CreateResource[*ApiResource](dag, ApiResourceCreateParams{
				AppName: params.AppName,
				Refs:    params.Refs,
				Path:    params.Path,
				ApiName: params.ApiName,
			})
			if err != nil {
				return err
			}
			method.Resource = parentResource
			dag.AddDependency(parentResource, method)
		}
	}
	return nil
}

type ApiDeploymentCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi
// is correlated to the constructs which required its creation.
func (deployment *ApiDeployment) Create(dag *construct.ResourceGraph, params ApiDeploymentCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	deployment.Name = name
	deployment.ConstructRefs = params.Refs.Clone()
	if deployment.Triggers == nil {
		deployment.Triggers = make(map[string]string)
	}
	existingDeployment := dag.GetResource(deployment.Id())
	if existingDeployment != nil {
		graphDeployment := existingDeployment.(*ApiDeployment)
		graphDeployment.ConstructRefs.AddAll(params.Refs)
	} else {
		dag.AddResource(deployment)
	}
	return nil
}

type ApiStageCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

// Create takes in an all necessary parameters to generate the RestApi name and ensure that the RestApi
// is correlated to the constructs which required its creation.
func (stage *ApiStage) Create(dag *construct.ResourceGraph, params ApiStageCreateParams) error {

	name := apiResourceSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	stage.Name = name
	stage.ConstructRefs = params.Refs.Clone()

	existingResource := dag.GetResource(stage.Id())
	if existingResource != nil {
		graphResource := existingResource.(*ApiStage)
		graphResource.ConstructRefs.AddAll(params.Refs)
		return nil
	} else {
		dag.AddResource(stage)
	}
	return nil
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (api *RestApi) BaseConstructRefs() construct.BaseConstructSet {
	return api.ConstructRefs
}

// Id returns the id of the cloud resource
func (api *RestApi) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_REST_TYPE,
		Name:     api.Name,
	}
}

func (api *RestApi) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (res *ApiResource) BaseConstructRefs() construct.BaseConstructSet {
	return res.ConstructRefs
}

// Id returns the id of the cloud resource
func (res *ApiResource) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_RESOURCE_TYPE,
		Name:     res.Name,
	}
}

func (res *ApiResource) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (method *ApiMethod) BaseConstructRefs() construct.BaseConstructSet {
	return method.ConstructRefs
}

// Id returns the id of the cloud resource
func (method *ApiMethod) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_METHOD_TYPE,
		Name:     method.Name,
	}
}

func (method *ApiMethod) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (link *VpcLink) BaseConstructRefs() construct.BaseConstructSet {
	return link.ConstructRefs
}

// Id returns the id of the cloud resource
func (res *VpcLink) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     VPC_LINK_TYPE,
		Name:     res.Name,
	}
}

func (link *VpcLink) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (integration *ApiIntegration) BaseConstructRefs() construct.BaseConstructSet {
	return integration.ConstructRefs
}

// Id returns the id of the cloud resource
func (integration *ApiIntegration) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_INTEGRATION_TYPE,
		Name:     integration.Name,
	}
}
func (integration *ApiIntegration) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   false,
		RequiresNoDownstream: false,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (deployment *ApiDeployment) BaseConstructRefs() construct.BaseConstructSet {
	return deployment.ConstructRefs
}

// Id returns the id of the cloud resource
func (deployment *ApiDeployment) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_DEPLOYMENT_TYPE,
		Name:     deployment.Name,
	}
}

func (deployment *ApiDeployment) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   false,
		RequiresNoDownstream: false,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (stage *ApiStage) BaseConstructRefs() construct.BaseConstructSet {
	return stage.ConstructRefs
}

// Id returns the id of the cloud resource
func (stage *ApiStage) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     API_GATEWAY_STAGE_TYPE,
		Name:     stage.Name,
	}
}

func (stage *ApiStage) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: false,
	}
}
