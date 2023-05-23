package knowledgebase

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var ApiGatewayKB = knowledgebase.EdgeKB{
	knowledgebase.NewEdge[*resources.ApiDeployment, *resources.RestApi]():  {},
	knowledgebase.NewEdge[*resources.ApiStage, *resources.RestApi]():       {},
	knowledgebase.NewEdge[*resources.ApiStage, *resources.ApiDeployment](): {},
	knowledgebase.NewEdge[*resources.ApiMethod, *resources.RestApi]():      {},
	knowledgebase.NewEdge[*resources.ApiDeployment, *resources.ApiMethod](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deployment := source.(*resources.ApiDeployment)
			if deployment.Triggers == nil {
				deployment.Triggers = make(map[string]string)
			}
			deployment.Triggers[dest.Id().Name] = dest.Id().Name
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.ApiDeployment, *resources.ApiIntegration](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deployment := source.(*resources.ApiDeployment)
			if deployment.Triggers == nil {
				deployment.Triggers = make(map[string]string)
			}
			deployment.Triggers[dest.Id().Name] = dest.Id().Name
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.ApiIntegration, *resources.RestApi](): {
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.LambdaFunction{})},
	},
	knowledgebase.NewEdge[*resources.ApiResource, *resources.ApiResource]():    {},
	knowledgebase.NewEdge[*resources.ApiResource, *resources.RestApi]():        {},
	knowledgebase.NewEdge[*resources.ApiMethod, *resources.ApiResource]():      {},
	knowledgebase.NewEdge[*resources.ApiIntegration, *resources.ApiResource](): {},
	knowledgebase.NewEdge[*resources.ApiIntegration, *resources.ApiMethod]():   {},
	knowledgebase.NewEdge[*resources.ApiIntegration, *resources.LambdaFunction](): {
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			integration := source.(*resources.ApiIntegration)
			// This isnt an instance of an expanded path, rather an existing edge so ignore
			if integration.Name != "" {
				return nil
			}
			function := dest.(*resources.LambdaFunction)
			restApi, ok := data.Source.(*resources.RestApi)
			refs := core.DedupeAnnotationKeys(append(function.ConstructsRef, restApi.ConstructsRef...))
			if !ok {
				return fmt.Errorf("source of lambda to api integration expansion must be a rest api resource")
			}
			if len(data.Routes) == 0 {
				return fmt.Errorf("there are no routes to expand the edge for lambda to api integration")
			}

			for _, route := range data.Routes {
				var err error
				integration, err = core.CreateResource[*resources.ApiIntegration](dag, resources.ApiIntegrationCreateParams{
					AppName:    data.AppName,
					Refs:       refs,
					Path:       route.Path,
					ApiName:    restApi.Name,
					HttpMethod: strings.ToUpper(string(route.Verb)),
				})
				if err != nil {
					return err
				}
				integration.IntegrationHttpMethod = "POST"
				integration.Type = "AWS_PROXY"
				integration.Uri = core.IaCValue{Resource: function, Property: resources.LAMBDA_INTEGRATION_URI_IAC_VALUE}
				segments := strings.Split(route.Path, "/")
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

				permission, err := core.CreateResource[*resources.LambdaPermission](dag, resources.LambdaPermissionCreateParams{
					Name: fmt.Sprintf("%s-%s", function.Name, integration.RestApi.Id()),
					Refs: refs,
				})
				if err != nil {
					return err
				}
				permission.Function = function

				for _, res := range dag.GetUpstreamResources(restApi) {
					switch resource := res.(type) {
					case *resources.ApiDeployment:
						dag.AddDependency(resource, integration.Method)
						dag.AddDependency(resource, integration)
					}
				}
				dag.AddDependenciesReflect(permission)
				dag.AddDependency(permission, integration.RestApi)
				dag.AddDependenciesReflect(integration)
			}
			return nil
		},
	},
	knowledgebase.NewEdge[*resources.LambdaPermission, *resources.RestApi](): {
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			permission := source.(*resources.LambdaPermission)
			api := dest.(*resources.RestApi)
			permission.Principal = "apigateway.amazonaws.com"
			permission.Action = "lambda:InvokeFunction"
			permission.Source = core.IaCValue{Resource: api, Property: resources.API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE}
			return nil
		},
	},
}
