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
	knowledgebase.EdgeBuilder[*resources.ApiMethod, *resources.RestApi]{},
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.ApiMethod]{
		Configure: func(deployment *resources.ApiDeployment, method *resources.ApiMethod, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Triggers == nil {
				deployment.Triggers = make(map[string]string)
			}
			deployment.Triggers[method.Id().Name] = method.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.ApiIntegration]{
		Configure: func(deployment *resources.ApiDeployment, integration *resources.ApiIntegration, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Triggers == nil {
				deployment.Triggers = make(map[string]string)
			}
			deployment.Triggers[integration.Id().Name] = integration.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.RestApi]{
		ValidDestinations: []core.Resource{&resources.LambdaFunction{}, &resources.TargetGroup{}},
	},
	knowledgebase.EdgeBuilder[*resources.ApiResource, *resources.ApiResource]{},
	knowledgebase.EdgeBuilder[*resources.ApiResource, *resources.RestApi]{},
	knowledgebase.EdgeBuilder[*resources.ApiMethod, *resources.ApiResource]{},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.ApiResource]{},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.ApiMethod]{},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LambdaFunction]{
		Expand: func(integration *resources.ApiIntegration, function *resources.LambdaFunction, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			// This isnt an instance of an expanded path, rather an existing edge so ignore
			if integration.Name != "" {
				return nil
			}
			restApi, ok := data.Source.(*resources.RestApi)
			refs := function.ConstructsRef.CloneWith(restApi.ConstructsRef)
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
					Refs:       refs.Clone(),
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
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LoadBalancer]{
		Expand: func(integration *resources.ApiIntegration, lb *resources.LoadBalancer, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			// This isnt an instance of an expanded path, rather an existing edge so ignore
			if integration.Name != "" {
				return nil
			}
			restApi, ok := data.Source.(*resources.RestApi)
			refs := core.AnnotationKeySet{}
			refs.AddAll(lb.ConstructsRef)
			refs.AddAll(restApi.ConstructsRef)
			if !ok {
				return fmt.Errorf("source of eks to api integration expansion must be a rest api resource")
			}
			tg, ok := data.Destination.(*resources.TargetGroup)
			if !ok {
				return fmt.Errorf("destination of eks to api integration expansion must be a target group")
			}
			if len(data.Routes) == 0 {
				return fmt.Errorf("there are no routes to expand the edge for lambda to api integration")
			}
			listener, err := core.CreateResource[*resources.Listener](dag, resources.ListenerCreateParams{
				AppName:     data.AppName,
				Refs:        tg.ConstructsRef,
				Name:        tg.Name,
				NetworkType: resources.PrivateSubnet,
			})
			if err != nil {
				return err
			}
			tg.Port = 3000
			tg.Protocol = "TCP"
			tg.TargetType = "ip"
			if listener.LoadBalancer == nil {
				return fmt.Errorf("no load balancer was generated for expansion from %s -> %s", data.Source.Id(), data.Destination.Id())
			}
			listener.LoadBalancer.Type = "network"
			listener.LoadBalancer.Scheme = "internal"
			listener.DefaultActions = []*resources.LBAction{{TargetGroupArn: core.IaCValue{Resource: tg, Property: resources.ARN_IAC_VALUE}, Type: "forward"}}
			listener.Port = 80
			listener.Protocol = "TCP"

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

				vpcLink := &resources.VpcLink{
					Target:        listener.LoadBalancer,
					ConstructsRef: listener.LoadBalancer.ConstructsRef,
				}
				integration.IntegrationHttpMethod = strings.ToUpper(string(route.Verb))
				integration.Type = "HTTP_PROXY"
				integration.Uri = core.IaCValue{Resource: listener.LoadBalancer, Property: resources.NLB_INTEGRATION_URI_IAC_VALUE}
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
				integration.ConnectionType = "VPC_LINK"
				integration.VpcLink = vpcLink
				for _, res := range dag.GetUpstreamResources(restApi) {
					switch resource := res.(type) {
					case *resources.ApiDeployment:
						dag.AddDependency(resource, integration.Method)
						dag.AddDependency(resource, integration)
					}
				}
				dag.AddDependency(integration, listener.LoadBalancer)
				dag.AddDependenciesReflect(vpcLink)
				dag.AddDependenciesReflect(integration)
			}
			return nil
		},
		ValidDestinations: []core.Resource{&resources.TargetGroup{}},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaPermission, *resources.RestApi]{
		Configure: func(permission *resources.LambdaPermission, api *resources.RestApi, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			permission.Principal = "apigateway.amazonaws.com"
			permission.Action = "lambda:InvokeFunction"
			permission.Source = core.IaCValue{Resource: api, Property: resources.API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.VpcLink]{},
	knowledgebase.EdgeBuilder[*resources.VpcLink, *resources.LoadBalancer]{},
)
