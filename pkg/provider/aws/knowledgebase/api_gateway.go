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
		ReverseDirection: true,
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
			refs := core.BaseConstructSetOf(function, restApi)
			if !ok {
				return fmt.Errorf("source of lambda to api integration expansion must be a rest api resource")
			}
			if len(data.Routes) == 0 {
				data.Routes = append(data.Routes, core.Route{Path: fmt.Sprintf("/%s/*", function.Name), Verb: "ANY"})
			}

			err := createRoutesForIntegration(data.AppName, data.Routes, refs, dag, nil, restApi, &resources.AwsResourceValue{ResourceVal: function, PropertyVal: resources.LAMBDA_INTEGRATION_URI_IAC_VALUE})
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

			if lb.Name == "" || lb == nil {
				var err error
				lb, err = core.CreateResource[*resources.LoadBalancer](dag, resources.LoadBalancerCreateParams{
					AppName: data.AppName,
					Refs:    core.BaseConstructSetOf(integration),
					Name:    integration.Name,
				})
				if err != nil {
					return err
				}
			}

			if integration.Name != "" {
				return nil
			}
			if len(data.Routes) == 0 {
				data.Routes = append(data.Routes, core.Route{Path: fmt.Sprintf("/%s/*", lb.Name), Verb: "ANY"})
			}

			restApi, ok := data.Source.(*resources.RestApi)
			refs := core.BaseConstructSetOf(restApi)
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
					Refs:    core.BaseConstructSetOf(restApi, ecsService),
					Name:    ecsService.Name,
				})
				tg.Protocol = "TCP"
				tg.TargetType = "ip"
				if err != nil {
					return err
				}
				if ecsService.TaskDefinition == nil {
					return fmt.Errorf("task definition is not ready")
				}
				ecsService.LoadBalancers = append(ecsService.LoadBalancers, resources.EcsServiceLoadBalancerConfig{
					ContainerName:  ecsService.TaskDefinition.Name,
					ContainerPort:  3000,
					TargetGroupArn: &resources.AwsResourceValue{ResourceVal: tg, PropertyVal: resources.ARN_IAC_VALUE},
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
					Refs:    core.BaseConstructSetOf(instance),
					Name:    instance.Name,
				})
				if err != nil {
					return err
				}
				tg.Protocol = "HTTPS"
				tg.TargetType = "instance"
				dag.AddDependency(tg, instance)
			}

			if tg != nil {
				tg.Port = 3000
			}

			var vpcLink *resources.VpcLink
			if isEc2Dest || isTgDest || isEcsDest {
				listener, err := core.CreateResource[*resources.Listener](dag, resources.ListenerCreateParams{
					AppName:     data.AppName,
					Refs:        core.BaseConstructSetOf(tg),
					Name:        tg.Name,
					NetworkType: resources.PrivateSubnet,
				})
				if err != nil {
					return err
				}
				listener.LoadBalancer = lb
				dag.AddDependency(listener, lb)
				listener.LoadBalancer.Type = "network"
				listener.LoadBalancer.Scheme = "internal"
				listener.DefaultActions = []*resources.LBAction{{TargetGroupArn: &resources.AwsResourceValue{ResourceVal: tg, PropertyVal: resources.ARN_IAC_VALUE}, Type: "forward"}}
				listener.Port = 80
				listener.Protocol = tg.Protocol
				dag.AddDependency(listener, tg)

				vpcLink = &resources.VpcLink{
					Target:        listener.LoadBalancer,
					ConstructsRef: listener.LoadBalancer.ConstructsRef,
				}
			}

			err = createRoutesForIntegration(data.AppName, data.Routes, refs, dag, vpcLink, restApi, &resources.AwsResourceValue{ResourceVal: lb, PropertyVal: resources.NLB_INTEGRATION_URI_IAC_VALUE})
			if err != nil {
				return err
			}
			if vpcLink != nil {
				dag.AddDependenciesReflect(vpcLink)
			}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.LambdaPermission, *resources.RestApi]{
		Configure: func(permission *resources.LambdaPermission, api *resources.RestApi, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			permission.Principal = "apigateway.amazonaws.com"
			permission.Action = "lambda:InvokeFunction"
			permission.Source = &resources.AwsResourceValue{ResourceVal: api, PropertyVal: resources.API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE}
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.VpcLink]{},
	knowledgebase.EdgeBuilder[*resources.VpcLink, *resources.LoadBalancer]{},
)

func createRoutesForIntegration(appName string, routes []core.Route, refs core.BaseConstructSet, dag *core.ResourceGraph, vpcLink *resources.VpcLink, restApi *resources.RestApi, uri *resources.AwsResourceValue) error {
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
