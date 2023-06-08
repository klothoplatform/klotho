package knowledgebase

import (
	"errors"
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
			deployment.Triggers[method.Id().Name] = method.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiDeployment, *resources.ApiIntegration]{
		Configure: func(deployment *resources.ApiDeployment, integration *resources.ApiIntegration, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deployment.Triggers[integration.Id().Name] = integration.Id().Name
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.RestApi]{
		ReverseDirection:  true,
		ValidDestinations: []core.Resource{&resources.LambdaFunction{}, &resources.TargetGroup{}, &resources.Ec2Instance{}, &resources.EcsService{}},
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

			err := createRoutesForIntegration(data.AppName, data.Routes, refs, dag, nil, restApi, core.IaCValue{Resource: function, Property: resources.LAMBDA_INTEGRATION_URI_IAC_VALUE})
			if err != nil {
				return err
			}
			permission, err := core.CreateResource[*resources.LambdaPermission](dag, resources.LambdaPermissionCreateParams{
				Name: fmt.Sprintf("%s-%s", function.Name, restApi.Id()),
				Refs: refs,
			})
			if err != nil {
				return err
			}
			permission.Function = function
			dag.AddDependency(permission, restApi)
			dag.AddDependency(permission, function)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LoadBalancer]{
		Expand: func(integration *resources.ApiIntegration, lb *resources.LoadBalancer, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			// This isnt an instance of an expanded path, rather an existing edge so ignore
			if integration.Name != "" {
				return nil
			}
			if len(data.Routes) == 0 {
				return fmt.Errorf("there are no routes to expand the edge for eks to api integration")
			}

			restApi, ok := data.Source.(*resources.RestApi)
			refs := restApi.ConstructsRef.Clone()
			if !ok {
				return fmt.Errorf("source of eks to api integration expansion must be a rest api resource")
			}

			var tg *resources.TargetGroup
			var err error
			isTgDest := false
			ecsService, isEcsDest := data.Destination.(*resources.EcsService)
			if isEcsDest {
				tg, err = core.CreateResource[*resources.TargetGroup](dag, resources.TargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    ecsService.ConstructsRef.CloneWith(restApi.ConstructsRef),
					Name:    ecsService.Name,
				})
				tg.Protocol = "TCP"
				tg.TargetType = "ip"
				if err != nil {
					return err
				}
				ecsService.LoadBalancers = append(ecsService.LoadBalancers, resources.EcsServiceLoadBalancerConfig{
					ContainerName:  ecsService.TaskDefinition.Name,
					ContainerPort:  3000,
					TargetGroupArn: core.IaCValue{Resource: tg, Property: resources.ARN_IAC_VALUE},
				})

				dag.AddDependency(ecsService, tg)
				dag.AddDependenciesReflect(ecsService)
			} else if tg, isTgDest = data.Destination.(*resources.TargetGroup); isTgDest {
				tg.Protocol = "TCP"
				tg.TargetType = "ip"
			}

			instance, isEc2Dest := data.Destination.(*resources.Ec2Instance)
			if isEc2Dest {
				tg, err = core.CreateResource[*resources.TargetGroup](dag, resources.TargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    instance.ConstructsRef.Clone(),
					Name:    instance.Name,
				})
				if err != nil {
					return err
				}
				tg.Protocol = "HTTPS"
				tg.TargetType = "instance"
				dag.AddDependency(tg, instance)
			}
			tg.Port = 3000

			if !isEc2Dest && !isTgDest && !isEcsDest {
				return fmt.Errorf("destination of api integration -> load balancer expansion must be a target group, ecs service, or ec2 instance, but got %T", data.Destination)
			}

			listener, err := core.CreateResource[*resources.Listener](dag, resources.ListenerCreateParams{
				AppName:     data.AppName,
				Refs:        tg.ConstructsRef.Clone(),
				Name:        tg.Name,
				NetworkType: resources.PrivateSubnet,
			})
			if err != nil {
				return err
			}
			if listener.LoadBalancer == nil {
				return fmt.Errorf("no load balancer was generated for expansion from %s -> %s", data.Source.Id(), data.Destination.Id())
			}
			listener.LoadBalancer.Type = "network"
			listener.LoadBalancer.Scheme = "internal"
			listener.DefaultActions = []*resources.LBAction{{TargetGroupArn: core.IaCValue{Resource: tg, Property: resources.ARN_IAC_VALUE}, Type: "forward"}}
			listener.Port = 80
			listener.Protocol = tg.Protocol
			dag.AddDependenciesReflect(listener)

			vpcLink := &resources.VpcLink{
				Target:        listener.LoadBalancer,
				ConstructsRef: listener.LoadBalancer.ConstructsRef,
			}

			err = createRoutesForIntegration(data.AppName, data.Routes, refs, dag, vpcLink, restApi, core.IaCValue{Resource: listener.LoadBalancer, Property: resources.NLB_INTEGRATION_URI_IAC_VALUE})
			if err != nil {
				return err
			}
			dag.AddDependenciesReflect(vpcLink)
			return nil
		},
		ValidDestinations: []core.Resource{&resources.TargetGroup{}, &resources.Ec2Instance{}, &resources.EcsService{}},
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

func createRoutesForIntegration(appName string, routes []core.Route, refs core.AnnotationKeySet, dag *core.ResourceGraph, vpcLink *resources.VpcLink, restApi *resources.RestApi, uri core.IaCValue) error {
	var merr error
	for _, route := range routes {
		integration, err := core.CreateResource[*resources.ApiIntegration](dag, resources.ApiIntegrationCreateParams{
			AppName:    appName,
			Refs:       refs,
			Path:       route.Path,
			ApiName:    restApi.Name,
			HttpMethod: strings.ToUpper(string(route.Verb)),
		})
		if err != nil {
			merr = errors.Join(merr, err)
			continue
		}

		if vpcLink != nil {
			integration.IntegrationHttpMethod = strings.ToUpper(string(route.Verb))
			integration.Type = "HTTP_PROXY"
			integration.ConnectionType = "VPC_LINK"
			integration.VpcLink = vpcLink
		} else {
			integration.IntegrationHttpMethod = "POST"
			integration.Type = "AWS_PROXY"
		}

		integration.Uri = uri
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

		for _, res := range dag.GetUpstreamResources(restApi) {
			switch resource := res.(type) {
			case *resources.ApiDeployment:
				dag.AddDependency(resource, integration.Method)
				dag.AddDependency(resource, integration)
			}
		}
		dag.AddDependenciesReflect(integration)
	}
	return merr
}
