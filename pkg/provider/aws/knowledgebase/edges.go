package knowledgebase

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var AwsKB = knowledgebase.EdgeKB{
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaPermission{}), Destination: reflect.TypeOf(&resources.LambdaFunction{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			permission := source.(*resources.LambdaPermission)
			function := dest.(*resources.LambdaFunction)
			if permission.Function != nil && permission.Function != function {
				return fmt.Errorf("cannot configure edge %s -> %s, permission already tied to function %s", permission.Id(), function.Id(), permission.Function.Id())
			}
			permission.Function = function
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.LambdaFunction{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.Subnet{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			lambda.Role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"})
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.Subnet{}), reflect.TypeOf(&resources.Vpc{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.SecurityGroup{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.SecurityGroup{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.RdsInstance{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			instance := dest.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = instance.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = instance.SecurityGroups
			}
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			instance := dest.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id(), instance.Id(), lambda.Id())
			}
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name), core.DedupeAnnotationKeys(append(lambda.ConstructsRef, instance.ConstructsRef...)), instance.GetConnectionPolicyDocument())
			lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)

			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: instance, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.RdsProxy{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			instance := data.Destination.(*resources.RdsInstance)
			if len(lambda.Subnets) == 0 {
				lambda.Subnets = instance.SubnetGroup.Subnets
			}
			if len(lambda.SecurityGroups) == 0 {
				lambda.SecurityGroups = instance.SecurityGroups
			}
			dag.AddDependenciesReflect(lambda)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			proxy := dest.(*resources.RdsProxy)
			if len(lambda.Subnets) == 0 {
				return fmt.Errorf("unable to expand edge [%s -> %s]: lambda function [%s] is not in a VPC",
					lambda.Id().String(), proxy.Id().String(), lambda.Id().String())
			}

			upstreamResources := dag.GetUpstreamResources(proxy)
			for _, res := range upstreamResources {
				if tg, ok := res.(*resources.RdsProxyTargetGroup); ok {
					for _, tgUpstream := range dag.GetDownstreamResources(tg) {
						if instance, ok := tgUpstream.(*resources.RdsInstance); ok {
							inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-connectionpolicy", instance.Name),
								core.DedupeAnnotationKeys(append(lambda.ConstructsRef, instance.ConstructsRef...)), instance.GetConnectionPolicyDocument())
							lambda.Role.InlinePolicies = append(lambda.Role.InlinePolicies, inlinePol)
							dag.AddDependency(lambda.Role, instance)
						}
					}
				}
			}
			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: proxy, Property: env.GetValue()}
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), Destination: reflect.TypeOf(&resources.RdsInstance{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			instance := dest.(*resources.RdsInstance)
			targetGroup := source.(*resources.RdsProxyTargetGroup)
			if targetGroup.Name == "" {
				proxyTargetGroup, err := core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    instance.ConstructsRef,
					Name:    instance.Name,
				})
				if err != nil {
					return err
				}
				proxyTargetGroup.RdsInstance = instance
				dag.AddDependency(proxyTargetGroup, instance)
			}
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			instance := dest.(*resources.RdsInstance)
			targetGroup := source.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsInstance == nil {
				targetGroup.RdsInstance = instance
			} else if targetGroup.RdsInstance.Name != instance.Name {
				return fmt.Errorf("target group, %s, has  Destination instance, %s, but internal property is set Destination a different instance %s", targetGroup.Name, instance.Name, targetGroup.RdsInstance.Name)
			}
			if targetGroup.RdsProxy != nil {
				secret := targetGroup.RdsProxy.Auths[0].SecretArn.Resource.(*resources.Secret)
				for _, res := range dag.GetUpstreamResources(secret) {
					if secretVersion, ok := res.(*resources.SecretVersion); ok {
						secretVersion.Path = instance.CredentialsPath
					}
				}
			}

			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxyTargetGroup{}), Destination: reflect.TypeOf(&resources.RdsProxy{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			proxy := dest.(*resources.RdsProxy)
			targetGroup := source.(*resources.RdsProxyTargetGroup)
			destination := data.Destination.(*resources.RdsInstance)
			if proxy.Name == "" {
				var err error
				proxy, err = core.CreateResource[*resources.RdsProxy](dag, resources.RdsProxyCreateParams{
					AppName: data.AppName,
					Refs:    targetGroup.ConstructsRef,
					Name:    destination.Name,
				})
				if err != nil {
					return err
				}
				dag.AddDependencyWithData(data.Source, proxy, data)
			}

			if targetGroup.Name == "" {
				proxyTargetGroup, err := core.CreateResource[*resources.RdsProxyTargetGroup](dag, resources.RdsProxyTargetGroupCreateParams{
					AppName: data.AppName,
					Refs:    proxy.ConstructsRef,
					Name:    destination.Name,
				})
				if err != nil {
					return err
				}
				proxyTargetGroup.RdsProxy = proxy
				dag.AddDependency(proxyTargetGroup, proxy)
			}
			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName: data.AppName,
				Refs:    core.DedupeAnnotationKeys(append(source.KlothoConstructRef(), dest.KlothoConstructRef()...)),
				Name:    proxy.Name,
			})
			if err != nil {
				return err
			}
			proxy.Auths = append(proxy.Auths, &resources.ProxyAuth{
				AuthScheme: "SECRETS",
				IamAuth:    "DISABLED",
				SecretArn:  core.IaCValue{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE},
			})
			dag.AddDependency(proxy, secretVersion)

			secretPolicy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    fmt.Sprintf("%s-ormsecretpolicy", proxy.Name),
				Refs:    proxy.ConstructsRef,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(secretPolicy, secretVersion)
			dag.AddDependency(proxy.Role, secretPolicy)
			dag.AddDependenciesReflect(proxy)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			proxy := dest.(*resources.RdsProxy)
			targetGroup := source.(*resources.RdsProxyTargetGroup)

			if targetGroup.RdsProxy == nil {
				targetGroup.RdsProxy = proxy
			} else if targetGroup.RdsProxy.Name != proxy.Name {
				return fmt.Errorf("target group, %s, has destination proxy, %s, but internal property is set Destination a different proxy %s", targetGroup.Name, proxy.Name, targetGroup.RdsProxy.Name)
			}
			if targetGroup.RdsInstance != nil {
				secret := proxy.Auths[0].SecretArn.Resource.(*resources.Secret)
				for _, res := range dag.GetUpstreamResources(secret) {
					if secretVersion, ok := res.(*resources.SecretVersion); ok {
						secretVersion.Path = targetGroup.RdsInstance.CredentialsPath
					}
				}
			}
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsInstance{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxy{}), Destination: reflect.TypeOf(&resources.SecretVersion{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			proxy := dest.(*resources.RdsProxy)
			secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
				AppName: data.AppName,
				Refs:    core.DedupeAnnotationKeys(append(source.KlothoConstructRef(), dest.KlothoConstructRef()...)),
				Name:    proxy.Name,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(proxy, secretVersion)

			secretPolicy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
				AppName: data.AppName,
				Name:    fmt.Sprintf("%s-ormsecretpolicy", proxy.Name),
				Refs:    proxy.ConstructsRef,
			})
			if err != nil {
				return err
			}
			dag.AddDependency(secretPolicy, secretVersion)
			dag.AddDependency(proxy.Role, secretPolicy)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			secretVersion := dest.(*resources.SecretVersion)
			secretVersion.Type = "string"
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxy{}), Destination: reflect.TypeOf(&resources.IamRole{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role := dest.(*resources.IamRole)
			role.AssumeRolePolicyDoc = resources.RDS_ASSUME_ROLE_POLICY
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.IamPolicy{}), Destination: reflect.TypeOf(&resources.SecretVersion{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy := source.(*resources.IamPolicy)
			secretVersion := dest.(*resources.SecretVersion)
			secretPolicyDoc := resources.CreateAllowPolicyDocument([]string{"secretsmanager:GetSecretValue"}, []core.IaCValue{{Resource: secretVersion.Secret, Property: resources.ARN_IAC_VALUE}})
			if policy.Policy == nil {
				policy.Policy = secretPolicyDoc
			} else {
				policy.Policy.Statement = append(policy.Policy.Statement, secretPolicyDoc.Statement...)
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.IamRole{}), Destination: reflect.TypeOf(&resources.IamPolicy{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			policy := dest.(*resources.IamPolicy)
			role := source.(*resources.IamRole)
			role.AddManagedPolicy(core.IaCValue{Resource: policy, Property: resources.ARN_IAC_VALUE})
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsSubnetGroup{}), Destination: reflect.TypeOf(&resources.Subnet{})}: knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsInstance{}), Destination: reflect.TypeOf(&resources.RdsSubnetGroup{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RdsSubnetGroup{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsInstance{}), Destination: reflect.TypeOf(&resources.SecurityGroup{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.SecurityGroup{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxy{}), Destination: reflect.TypeOf(&resources.SecurityGroup{})}: knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxy{}), Destination: reflect.TypeOf(&resources.Subnet{})}:        knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RdsProxy{}), Destination: reflect.TypeOf(&resources.Secret{})}:        knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.SecretVersion{}), Destination: reflect.TypeOf(&resources.Secret{})}:   knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.IamRole{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role := dest.(*resources.IamRole)
			role.AssumeRolePolicyDoc = resources.LAMBDA_ASSUMER_ROLE_POLICY
			role.AddAwsManagedPolicies([]string{"arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"})
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.IamRole{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.EcrImage{})}: knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.LogGroup{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup := dest.(*resources.LogGroup)
			function := source.(*resources.LambdaFunction)
			logGroup.LogGroupName = fmt.Sprintf("/aws/lambda/%s", function.Name)
			logGroup.RetentionInDays = 5
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.EcrImage{}), Destination: reflect.TypeOf(&resources.EcrRepository{})}: knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.SecurityGroup{}), Destination: reflect.TypeOf(&resources.Vpc{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			sg := source.(*resources.SecurityGroup)
			vpc := dest.(*resources.Vpc)
			vpcIngressRule := resources.SecurityGroupRule{
				Description: "Allow ingress traffic from ip addresses within the vpc",
				CidrBlocks: []core.IaCValue{
					{Resource: vpc, Property: resources.CIDR_BLOCK_IAC_VALUE},
				},
				FromPort: 0,
				Protocol: "-1",
				ToPort:   0,
			}
			selfIngressRule := resources.SecurityGroupRule{
				Description: "Allow ingress traffic from within the same security group",
				FromPort:    0,
				Protocol:    "-1",
				ToPort:      0,
				Self:        true,
			}
			sg.IngressRules = append(sg.IngressRules, vpcIngressRule, selfIngressRule)

			allOutboundRule := resources.SecurityGroupRule{
				Description: "Allows all outbound IPv4 traffic.",
				FromPort:    0,
				Protocol:    "-1",
				ToPort:      0,
				CidrBlocks: []core.IaCValue{
					{Property: "0.0.0.0/0"},
				},
			}
			sg.EgressRules = append(sg.EgressRules, allOutboundRule)
			return nil
		},
	},

	//Networking Edges
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.NatGateway{}), Destination: reflect.TypeOf(&resources.Subnet{})}:    knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.NatGateway{}), Destination: reflect.TypeOf(&resources.ElasticIp{})}: knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RouteTable{}), Destination: reflect.TypeOf(&resources.Subnet{})}:    knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RouteTable{}), Destination: reflect.TypeOf(&resources.NatGateway{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			routeTable := source.(*resources.RouteTable)
			nat := dest.(*resources.NatGateway)
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					return fmt.Errorf("route table %s already has route for 0.0.0.0/0", routeTable.Name)
				}
			}
			routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", NatGatewayId: core.IaCValue{Resource: nat, Property: resources.ID_IAC_VALUE}})
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RouteTable{}), Destination: reflect.TypeOf(&resources.InternetGateway{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			igw := dest.(*resources.InternetGateway)
			routeTable := source.(*resources.RouteTable)
			for _, route := range routeTable.Routes {
				if route.CidrBlock == "0.0.0.0/0" {
					return fmt.Errorf("route table %s already has route for 0.0.0.0/0", routeTable.Name)
				}
			}
			routeTable.Routes = append(routeTable.Routes, &resources.RouteTableRoute{CidrBlock: "0.0.0.0/0", GatewayId: core.IaCValue{Resource: igw, Property: resources.ID_IAC_VALUE}})
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.RouteTable{}), Destination: reflect.TypeOf(&resources.Vpc{})}:           knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.InternetGateway{}), Destination: reflect.TypeOf(&resources.Vpc{})}:      knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.Subnet{}), Destination: reflect.TypeOf(&resources.Vpc{})}:               knowledgebase.EdgeDetails{},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.Subnet{}), Destination: reflect.TypeOf(&resources.AvailabilityZones{})}: knowledgebase.EdgeDetails{},

	// Expose Api Gateway Routes
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiDeployment{}), Destination: reflect.TypeOf(&resources.RestApi{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RestApi{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiStage{}), Destination: reflect.TypeOf(&resources.RestApi{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RestApi{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiStage{}), Destination: reflect.TypeOf(&resources.ApiDeployment{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiDeployment{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiMethod{}), Destination: reflect.TypeOf(&resources.RestApi{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RestApi{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiDeployment{}), Destination: reflect.TypeOf(&resources.ApiMethod{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deployment := source.(*resources.ApiDeployment)
			if deployment.Triggers == nil {
				deployment.Triggers = make(map[string]string)
			}
			deployment.Triggers[dest.Id().Name] = dest.Id().Name
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiDeployment{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiDeployment{}), Destination: reflect.TypeOf(&resources.ApiIntegration{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			deployment := source.(*resources.ApiDeployment)
			if deployment.Triggers == nil {
				deployment.Triggers = make(map[string]string)
			}
			deployment.Triggers[dest.Id().Name] = dest.Id().Name
			return nil
		},
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiDeployment{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiIntegration{}), Destination: reflect.TypeOf(&resources.RestApi{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.LambdaFunction{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiResource{}), Destination: reflect.TypeOf(&resources.RestApi{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RestApi{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiResource{}), Destination: reflect.TypeOf(&resources.ApiResource{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiResource{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiMethod{}), Destination: reflect.TypeOf(&resources.ApiResource{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiResource{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiIntegration{}), Destination: reflect.TypeOf(&resources.ApiResource{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiResource{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiIntegration{}), Destination: reflect.TypeOf(&resources.ApiMethod{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.ApiMethod{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.ApiIntegration{}), Destination: reflect.TypeOf(&resources.LambdaFunction{})}: knowledgebase.EdgeDetails{
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
				permission.Principal = "apigateway.amazonaws.com"
				permission.Action = "lambda:InvokeFunction"
				permission.Source = core.IaCValue{Resource: integration.RestApi, Property: resources.API_GATEWAY_EXECUTION_CHILD_RESOURCES_IAC_VALUE}
				for _, res := range dag.GetUpstreamResources(restApi) {
					switch resource := res.(type) {
					case *resources.ApiDeployment:
						dag.AddDependency(resource, integration.Method)
						dag.AddDependency(resource, integration)
					}
				}
				dag.AddDependenciesReflect(permission)
				dag.AddDependenciesReflect(integration)
			}
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaPermission{}), Destination: reflect.TypeOf(&resources.RestApi{})}: knowledgebase.EdgeDetails{
		ValidDestinations: []reflect.Type{reflect.TypeOf(&resources.RestApi{})},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.IamRole{}), Destination: reflect.TypeOf(&resources.DynamodbTable{})}: knowledgebase.EdgeDetails{
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			role := source.(*resources.IamRole)
			table := dest.(*resources.DynamodbTable)

			actions := []string{"dynamodb:*"}
			policyResources := []core.IaCValue{
				{Resource: table, Property: resources.ARN_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_BACKUP_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_INDEX_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_EXPORT_IAC_VALUE},
				{Resource: table, Property: resources.DYNAMODB_TABLE_STREAM_IAC_VALUE},
			}
			doc := resources.CreateAllowPolicyDocument(actions, policyResources)
			inlinePol := resources.NewIamInlinePolicy(fmt.Sprintf("%s-dynamodb-policy", table.Name), core.DedupeAnnotationKeys(append(role.ConstructsRef, table.ConstructsRef...)), doc)
			role.InlinePolicies = append(role.InlinePolicies, inlinePol)
			return nil
		},
	},
	knowledgebase.Edge{Source: reflect.TypeOf(&resources.LambdaFunction{}), Destination: reflect.TypeOf(&resources.DynamodbTable{})}: knowledgebase.EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			table := dest.(*resources.DynamodbTable)
			dag.AddDependency(lambda.Role, table)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			lambda := source.(*resources.LambdaFunction)
			table := dest.(*resources.DynamodbTable)

			for _, env := range data.EnvironmentVariables {
				lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Resource: table, Property: env.GetValue()}
			}
			return nil
		},
	},
}
