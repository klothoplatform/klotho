package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var ApiGatewayKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.ApiMethod]{
		Configure: func(deployment *resources.ApiDeployment, method *resources.ApiMethod, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if method == nil || deployment == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", method.Id())
			}
			if deployment.Triggers == nil {
				deployment.Triggers = map[string]string{}
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
			if deployment.Triggers == nil {
				deployment.Triggers = map[string]string{}
			}
			deployment.Triggers[integration.Id().Name] = integration.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LambdaFunction]{
		Configure: func(integration *resources.ApiIntegration, function *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if integration.RestApi == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
			}
			return configureIntegration(integration, dag)
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LoadBalancer]{
		Reuse: knowledgebase.Downstream,
		Configure: func(integration *resources.ApiIntegration, loadBalancer *resources.LoadBalancer, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if integration.Method == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
			}
			integration.IntegrationHttpMethod = strings.ToUpper(integration.Method.HttpMethod)
			return configureIntegration(integration, dag)
		},
	},
)

func configureIntegration(integration *resources.ApiIntegration, dag *core.ResourceGraph) error {

	if integration.RestApi == nil || integration.Method == nil {
		return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
	}

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
