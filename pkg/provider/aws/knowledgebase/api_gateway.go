package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var ApiGatewayKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.RestApi]{},
	knowledgebase.EdgeBuilder[*resources.ApiStage, *resources.RestApi]{},
	knowledgebase.EdgeBuilder[*resources.ApiStage, *resources.ApiDeployment]{},
	knowledgebase.EdgeBuilder[*resources.RestApi, *resources.ApiMethod]{
		DeploymentOrderReversed: true,
	},
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.ApiMethod]{
		Configure: func(deployment *resources.ApiDeployment, method *resources.ApiMethod, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if method == nil || deployment == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", method.Id())
			}
			deployment.Triggers[method.Id().Name] = method.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.ApiIntegration]{
		Configure: func(deployment *resources.ApiDeployment, integration *resources.ApiIntegration, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if integration == nil || deployment == nil {
				return fmt.Errorf("cannot configure edge %s", integration.Id())
			}
			deployment.Triggers[integration.Id().Name] = integration.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.RestApi, *resources.ApiIntegration]{
		DeploymentOrderReversed: true,
	},
	knowledgebase.EdgeBuilder[*resources.ApiResource, *resources.ApiResource]{},
	knowledgebase.EdgeBuilder[*resources.RestApi, *resources.ApiResource]{
		DeploymentOrderReversed: true,
	},
	knowledgebase.EdgeBuilder[*resources.ApiMethod, *resources.ApiResource]{},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.ApiResource]{},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.ApiMethod]{},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LambdaFunction]{
		Configure: func(integration *resources.ApiIntegration, function *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if integration.RestApi == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
			}
			integration.IntegrationHttpMethod = "POST"
			integration.Type = "AWS_PROXY"

			permission, err := core.CreateResource[*resources.LambdaPermission](dag, resources.LambdaPermissionCreateParams{
				Name: fmt.Sprintf("%s-%s", function.Name, integration.RestApi.Id()),
				Refs: core.BaseConstructSetOf(integration, function),
			})
			if err != nil {
				return err
			}
			permission.Function = function
			dag.AddDependency(permission, integration.RestApi)
			dag.AddDependency(permission, function)
			return configureIntegration(integration, dag, core.IaCValue{ResourceId: function.Id(), Property: resources.LAMBDA_INTEGRATION_URI_IAC_VALUE})
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LoadBalancer]{
		Reuse: knowledgebase.Downstream,
		Configure: func(integration *resources.ApiIntegration, loadBalancer *resources.LoadBalancer, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if integration.Method == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
			}
			vpcLink := &resources.VpcLink{
				Target:        loadBalancer,
				ConstructRefs: core.BaseConstructSetOf(loadBalancer, integration),
			}
			integration.IntegrationHttpMethod = strings.ToUpper(integration.Method.HttpMethod)
			integration.Type = "HTTP_PROXY"
			integration.ConnectionType = "VPC_LINK"
			integration.VpcLink = vpcLink
			dag.AddDependenciesReflect(vpcLink)
			return configureIntegration(integration, dag, core.IaCValue{ResourceId: loadBalancer.Id(), Property: resources.NLB_INTEGRATION_URI_IAC_VALUE})
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaPermission, *resources.RestApi]{
		Configure: func(permission *resources.LambdaPermission, api *resources.RestApi, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			permission.Principal = "apigateway.amazonaws.com"
			permission.Action = "lambda:InvokeFunction"
			permission.Source = core.IaCValue{ResourceId: api.Id(), Property: resources.API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.VpcLink]{},
	knowledgebase.EdgeBuilder[*resources.VpcLink, *resources.LoadBalancer]{},
)

func configureIntegration(integration *resources.ApiIntegration, dag *core.ResourceGraph, uri core.IaCValue) error {

	if integration.RestApi == nil || integration.Method == nil {
		return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
	}

	integration.Uri = uri
	segments := strings.Split(integration.Route, "/")
	methodRequestParams := map[string]bool{}
	integrationRequestParams := map[string]string{}
	for _, segment := range segments {
		if strings.Contains(segment, ":") {
			// We strip the pathParam of the : and * characters (which signal path parameters or wildcard routes) to be able to inject them into our method and integration request parameters
			pathParam := fmt.Sprintf("request.path.%s", segment)
			pathParam = strings.ReplaceAll(pathParam, ":", "")
			pathParam = strings.ReplaceAll(pathParam, "*", "")
			methodRequestParams[fmt.Sprintf("method.%s", pathParam)] = true
			integrationRequestParams[fmt.Sprintf("integration.%s", pathParam)] = fmt.Sprintf("method.%s", pathParam)
		}
	}
	integration.RequestParameters = integrationRequestParams
	integration.Method.RequestParameters = methodRequestParams

	for _, res := range dag.GetUpstreamResources(integration.RestApi) {
		switch resource := res.(type) {
		case *resources.ApiDeployment:
			dag.AddDependency(resource, integration.Method)
			dag.AddDependency(resource, integration)
		}
	}
	return nil
}
