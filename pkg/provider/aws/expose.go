package aws

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) GenerateExposeResources(gateway *core.Gateway, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	err := a.CreateRestApi(gateway, result, dag)
	return err
}

func (a *AWS) CreateRestApi(gateway *core.Gateway, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	var errs multierr.Error
	api := resources.NewRestApi(a.Config.AppName, gateway)
	dag.AddResource(api)
	api_references := []core.AnnotationKey{gateway.AnnotationKey}
	triggers := map[string]string{}

	resourceMap := map[string]*resources.ApiResource{}
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

		segments := strings.Split(route.Path, "/")
		currPathSegment := strings.Builder{}

		for _, segment := range segments {
			methodRequestParams := map[string]bool{}
			integrationRequestParams := map[string]string{}

			if strings.Contains(segment, ":") {
				pathParam := fmt.Sprintf("request.path.%s", segment)
				pathParam = strings.ReplaceAll(pathParam, ":", "")
				pathParam = strings.ReplaceAll(pathParam, "*", "")
				methodRequestParams[fmt.Sprintf("method.%s", pathParam)] = true
				integrationRequestParams[fmt.Sprintf("integration.%s", pathParam)] = fmt.Sprintf("method.%s", pathParam)
			}

			segment = convertPath(segment)
			currPathSegment.WriteString(fmt.Sprintf("%s/", segment))
			refs := []core.AnnotationKey{gateway.Provenance(), execUnit.Provenance()}
			resource, ok := resourceMap[segment]
			if !ok {
				resource = resources.NewApiResource(api, refs, segment)
				dag.AddResource(resource)
				dag.AddDependency2(resource, api)
				resourceMap[currPathSegment.String()] = resource
				triggers[resource.Name] = resource.Name
			}

			method := resources.NewApiMethod(resource, refs, strings.ToUpper(string(route.Verb)), methodRequestParams)
			dag.AddResource(method)
			dag.AddDependency2(method, resource)
			integration, err := a.createIntegration(method, execUnit, refs, route, dag)
			if err != nil {
				errs.Append(err)
				continue
			}

			triggers[integration.Name] = integration.Name
		}
	}
	api.ConstructsRef = api_references
	deployment := resources.NewApiDeployment(api, api_references, triggers)
	dag.AddResource(deployment)
	dag.AddDependency2(deployment, api)
	stage := resources.NewApiStage(deployment, "$default", api_references)
	dag.AddResource(stage)
	dag.AddDependency2(stage, deployment)
	return errs.ErrOrNil()
}

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
		lambdaPermission := resources.NewLambdaPermission(function, core.IaCValue{Resource: method.RestApi, Property: core.ARN_IAC_VALUE}, "apigateway.amazonaws.com", "lambda:InvokeFunction", refs)
		dag.AddResource(lambdaPermission)
		dag.AddDependency2(lambdaPermission, function)
		integration := resources.NewApiIntegration(method, refs, "POST", "AWS_PROXY", nil, core.IaCValue{Resource: function, Property: resources.LAMBDA_INTEGRATION_URI_IAC_VALUE})
		dag.AddResource(integration)
		dag.AddDependency2(integration, method)
		dag.AddDependency2(integration, function)
		return integration, nil
	default:
		return nil, errors.Errorf("Unrecognized integration type, %s, for api gateway", cfg.Type)
	}
}

func convertPath(path string) string {
	path = regexp.MustCompile(":([^/]+)").ReplaceAllString(path, "{$1}")
	path = regexp.MustCompile("[*]}").ReplaceAllString(path, "+}")
	path = regexp.MustCompile("//").ReplaceAllString(path, "/")
	return path
}
