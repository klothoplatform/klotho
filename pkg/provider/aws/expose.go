package aws

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

// GenerateExposeResources will create the necessary resources within AWS to support a Gateway construct and its dependencies.
func (a *AWS) GenerateExposeResources(gateway *core.Gateway, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	err := a.CreateRestApi(gateway, result, dag)
	return err
}

// CreateRestApi will create the the necessary resources within AWS to support a Gateway construct, of type RestAPI, and its dependencies, using the apigatewayv1 resources.
func (a *AWS) CreateRestApi(gateway *core.Gateway, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	var errs multierr.Error
	api := resources.NewRestApi(a.Config.AppName, gateway)
	dag.AddResource(api)
	api_references := []core.AnnotationKey{gateway.AnnotationKey}
	triggers := map[string]string{}

	resourceByCurrentSegment := map[string]*resources.ApiResource{}
	api_methods := []*resources.ApiMethod{}
	api_integrations := []*resources.ApiIntegration{}
	for _, route := range gateway.Routes {
		construct := result.GetConstruct(core.AnnotationKey{ID: route.ExecUnitName, Capability: annotation.ExecutionUnitCapability}.ToId())
		if construct == nil {
			errs = append(errs, errors.Errorf("Expected execution unit with id, %s, to exist", route.ExecUnitName))
			continue
		}
		execUnit, ok := construct.(*core.ExecutionUnit)
		if !ok {
			errs = append(errs, errors.Errorf("Expected construct with id, %s, to be an execution unit", construct.Id()))
			continue
		}
		api_references = append(api_references, execUnit.Provenance())
		routeTrigger := fmt.Sprintf("%s:%s:%s", route.ExecUnitName, route.Path, route.Verb)
		triggers[routeTrigger] = routeTrigger

		// We split our path by segments so that we can create a resource per segment as per api gateway v1
		segments := strings.Split(route.Path, "/")
		currPathSegment := strings.Builder{}
		refs := []core.AnnotationKey{gateway.Provenance(), execUnit.Provenance()}
		methodRequestParams := map[string]bool{}
		integrationRequestParams := map[string]string{}
		var parentResource *resources.ApiResource

		// This for loop ensures that an apigateway resource is created for every sub segment in the routes path
		// If one is not created, it will be generated with its parent being the resource which exists for the previous segment.
		for _, segment := range segments {

			// The root path is already created in api gw so we dont want to attempt to create an empty resource
			if segment == "" {
				continue
			}

			if strings.Contains(segment, ":") {
				// We strip the pathParam of the : and * characters (which signal path parameters or wildcard routes) to be able to inject them into our method and integration request parameters
				pathParam := fmt.Sprintf("request.path.%s", segment)
				pathParam = strings.ReplaceAll(pathParam, ":", "")
				pathParam = strings.ReplaceAll(pathParam, "*", "")
				methodRequestParams[fmt.Sprintf("method.%s", pathParam)] = true
				integrationRequestParams[fmt.Sprintf("integration.%s", pathParam)] = fmt.Sprintf("method.%s", pathParam)
			}
			segment = convertPath(segment)

			currPathSegment.WriteString(fmt.Sprintf("%s/", segment))
			currParentResource, ok := resourceByCurrentSegment[currPathSegment.String()]

			// If no resource exists at this segment, create a resource and set it as the parent
			if !ok {
				resource := resources.NewApiResource(currPathSegment.String(), api, refs, segment, parentResource)
				dag.AddDependenciesReflect(resource)
				resourceByCurrentSegment[currPathSegment.String()] = resource
				triggers[resource.Name] = resource.Name
				// If there is currently a parent segment, it would be in the route before the new segment, thus add a dependency
				if parentResource != nil {
					dag.AddDependency(resource, parentResource)
				}
				parentResource = resource
			} else {
				// The resource for this segment already exists, so just update the current parent
				parentResource = currParentResource
			}
		}

		// Now that we know there are resouces for the full path of this route, we can create the specific method for the route
		method := resources.NewApiMethod(parentResource, api, refs, strings.ToUpper(string(route.Verb)), methodRequestParams)
		dag.AddDependenciesReflect(method)
		api_methods = append(api_methods, method)
		if parentResource != nil {
			dag.AddDependency(method, parentResource)
		}
		integration, err := a.createIntegration(method, execUnit, refs, route, dag)
		if err != nil {
			errs.Append(err)
			continue
		}
		api_integrations = append(api_integrations, integration)
		triggers[integration.Name] = integration.Name

	}
	api.ConstructsRef = api_references
	deployment := resources.NewApiDeployment(api, api_references, triggers)
	dag.AddDependenciesReflect(deployment)
	for _, m := range api_methods {
		dag.AddDependency(deployment, m)
	}
	for _, integration := range api_integrations {
		dag.AddDependency(deployment, integration)
	}

	stage := resources.NewApiStage(deployment, "stage", api_references)
	dag.AddDependenciesReflect(stage)
	a.MapResourceDirectlyToConstruct(stage, gateway)
	a.MapResourceDirectlyToConstruct(api, gateway)
	return errs.ErrOrNil()
}

// createIntegration will create the the necessary resources within AWS to support a dependency between an expose construct and an execution unit.
func (a *AWS) createIntegration(method *resources.ApiMethod, unit *core.ExecutionUnit, refs []core.AnnotationKey, route core.Route, dag *core.ResourceGraph) (*resources.ApiIntegration, error) {
	cfg := a.Config.GetExecutionUnit(unit.ID)
	switch cfg.Type {
	case Lambda:
		constructResources, _ := a.GetResourcesDirectlyTiedToConstruct(unit)
		if len(constructResources) != 1 {
			return nil, errors.Errorf("Expected one resource to be tied to a lambda execution unit, %s, but found %s", unit.ID, strconv.Itoa(len(constructResources)))
		}
		function, ok := constructResources[0].(*resources.LambdaFunction)
		if !ok {
			return nil, errors.Errorf("Expected resource to be of type, lambda function, for execution unit, %s", unit.ID)
		}
		lambdaPermission := resources.NewLambdaPermission(function, core.IaCValue{Resource: method, Property: resources.ALL_RESOURCES_ARN_IAC_VALUE}, "apigateway.amazonaws.com", "lambda:InvokeFunction", refs)
		dag.AddDependenciesReflect(lambdaPermission)

		accountId := resources.NewAccountId()
		dag.AddDependency(lambdaPermission, accountId)

		integration := resources.NewApiIntegration(method, refs, "POST", "AWS_PROXY", nil, core.IaCValue{Resource: function, Property: resources.LAMBDA_INTEGRATION_URI_IAC_VALUE})
		dag.AddDependenciesReflect(integration)
		return integration, nil
	case kubernetes.KubernetesType:

		cluster, err := findUnitsCluster(unit, dag)
		if err != nil {
			return nil, err
		}
		err = cluster.InstallAlbController([]core.AnnotationKey{unit.AnnotationKey}, dag)
		if err != nil {
			return nil, err
		}
		nlb := dag.GetResource(resources.NewLoadBalancer(a.Config.AppName, unit.ID, nil, "internal", "network", nil, nil).Id())
		if nlb == nil {
			return nil, errors.Errorf("No nlb created for unit %s", unit.ID)
		}
		vpcLink := resources.NewVpcLink(nlb, refs)
		integration := resources.NewApiIntegration(method, refs, method.HttpMethod, "HTTP_PROXY", vpcLink, core.IaCValue{Resource: nlb, Property: resources.NLB_INTEGRATION_URI_IAC_VALUE})
		integration.Route = convertPath(route.Path)
		integration.ConnectionType = "VPC_LINK"
		dag.AddDependency(integration, nlb)
		dag.AddDependenciesReflect(vpcLink)
		dag.AddDependenciesReflect(integration)
		return integration, nil
	default:
		return nil, errors.Errorf("Unrecognized integration type, %s, for api gateway", cfg.Type)
	}
}

// convertPath will convert the path stored in our gateway construct into a path that is functionaliy the same within apigateway.
//
// The path will be minpulated so that:
// - any : characters will be removed and replaced the item with surrounding brackets, to signal this is a path parameter
// - any escaped / will turn into a singal /
// - any wildcard route will be propagated to the apigateway standard format
func convertPath(path string) string {
	path = regexp.MustCompile(":([^/]+)").ReplaceAllString(path, "{$1}")
	path = regexp.MustCompile("[*]}").ReplaceAllString(path, "+}")
	path = regexp.MustCompile("//").ReplaceAllString(path, "/")
	return path
}
