import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { CloudCCLib, ResourceKey, Resource } from '../deploylib'
import * as sha256 from 'simple-sha256'
import { LoadBalancerPlugin } from './load_balancing'

export interface Route {
    verb: string
    path: string
    execUnitName: string
}

export interface Gateway {
    Name: string
    Routes: Route[]
    Targets: any
}

export class ApiGateway {
    private readonly vpcLink: aws.apigatewayv2.VpcLink
    public readonly invokeUrls: pulumi.Output<string>[] = []
    private readonly execUnitToIntegration: Map<string, aws.apigatewayv2.Integration> = new Map<
        string,
        aws.apigatewayv2.Integration
    >()

    constructor(
        private readonly lib: CloudCCLib,
        private readonly lbPlugin: LoadBalancerPlugin,
        gateways: Gateway[]
    ) {
        if (lib.klothoVPC != undefined) {
            this.vpcLink = this.createVpcLink(
                lib.sgs,
                pulumi
                    .all([lib.publicSubnetIds || [], lib.privateSubnetIds || []])
                    .apply(([publicIds, privateIds]) => {
                        return [...publicIds, ...privateIds]
                    })
            )
        }

        gateways.forEach((gateway) => {
            //   if (gateway.ProtocolType === "websocket") {
            this.createWebSocketGateway(gateway)
            //   } else if (gateway.ProtocolType === "http") {
            //     this.createDockerBasedAPIGateway(gateway.Routes, gateway.Name)
            //   }
        })
    }

    createWebSocketApiGateway(providedName): aws.apigatewayv2.Api {
        return new aws.apigatewayv2.Api(`${this.lib.name}-${providedName}`, {
            protocolType: 'WEBSOCKET',
            routeSelectionExpression: `$request.body.action`,
        })
    }

    createVpcLink(
        securityGroupIds: pulumi.Output<string>[],
        subnetIds: pulumi.Output<string[]>
    ): aws.apigatewayv2.VpcLink {
        return new aws.apigatewayv2.VpcLink(`${this.lib.name}`, {
            securityGroupIds,
            subnetIds,
        })
    }

    createLambdaIntegration(
        gwName: string,
        api: aws.apigatewayv2.Api,
        invokeArn: pulumi.Output<string>,
        createVPC: boolean,
        verb: string,
        execUnitName: string
    ): aws.apigatewayv2.Integration {
        return new aws.apigatewayv2.Integration(
            `${this.lib.name}-${gwName}-${execUnitName}`,
            {
                apiId: api.id,
                integrationType: 'AWS_PROXY',
                connectionType: createVPC ? 'VPC_LINK' : 'INTERNET',
                contentHandlingStrategy: 'CONVERT_TO_TEXT',
                integrationMethod: verb,
                integrationUri: invokeArn,
            },
            {
                parent: api,
            }
        )
    }

    createPrivateIntegration(
        gwName: string,
        api: aws.apigatewayv2.Api,
        vpcLink: aws.apigatewayv2.VpcLink,
        integrationUri: pulumi.Output<string>,
        verb: string,
        execUnitName: string
    ): aws.apigatewayv2.Integration {
        return new aws.apigatewayv2.Integration(
            `${this.lib.name}-${gwName}-${execUnitName}`,
            {
                apiId: api.id,
                description: 'Example with a load balancer',
                integrationType: 'HTTP_PROXY',
                integrationUri,
                integrationMethod: verb,
                connectionId: '${stageVariables.vpcLinkId}',
                connectionType: 'VPC_LINK',
                passthroughBehavior: 'WHEN_NO_MATCH',
            },
            {
                dependsOn: [vpcLink],
                parent: api,
            }
        )
    }

    createRoute(
        gwName: string,
        api: aws.apigatewayv2.Api,
        route: string,
        integration: aws.apigatewayv2.Integration
    ): aws.apigatewayv2.Route {
        return new aws.apigatewayv2.Route(
            `${this.lib.name}-${gwName}-${route}`,
            {
                apiId: api.id,
                routeKey: route,
                target: pulumi.interpolate`integrations/${integration.id}`,
            },
            {
                parent: api,
            }
        )
    }

    createDeployment(gwName: string, api: aws.apigatewayv2.Api, routes: aws.apigatewayv2.Route[]) {
        return new aws.apigatewayv2.Deployment(
            `${this.lib.name}-${gwName}`,
            {
                apiId: api.id,
            },
            {
                dependsOn: routes,
                parent: api,
            }
        )
    }

    createStage(
        gwName: string,
        api: aws.apigatewayv2.Api,
        deployment: aws.apigatewayv2.Deployment,
        vpcLink: aws.apigatewayv2.VpcLink,
        stageName: string
    ) {
        return new aws.apigatewayv2.Stage(
            `${this.lib.name}-${gwName}-${stageName}`,
            {
                apiId: api.id,
                name: stageName,
                deploymentId: deployment.id,
                stageVariables: {
                    vpcLinkId: vpcLink.id,
                },
            },
            {
                dependsOn: [vpcLink],
                parent: api,
            }
        )
    }

    createWebSocketGateway(gateway: Gateway) {
        const gwName = gateway.Name.replace(/[^a-zA-Z0-9_-]/g, '-')
        const api: aws.apigatewayv2.Api = this.createWebSocketApiGateway(gwName)
        const apiRoutes: aws.apigatewayv2.Route[] = []
        const units = new Set<string>()
        gateway.Routes.forEach((gw) => units.add(gw.execUnitName))
        if (units.size > 1) {
            throw new Error('only one exec unit is supported for websocket API Gateway')
        }
        for (const route of gateway.Routes) {
            const execUnit = this.lib.resourceIdToResource.get(`${route.execUnitName}_exec_unit`)
            if (execUnit.type == 'fargate') {
                let integration: aws.apigatewayv2.Integration | undefined =
                    this.execUnitToIntegration.get(route.execUnitName)
                if (!integration) {
                    const nlb = this.lib.execUnitToNlb.get(route.execUnitName)!
                    const integrationUri = pulumi.interpolate`http://${
                        nlb.loadBalancer.dnsName
                    }${route.path.replace(/:([^/]+)/g, '{$1}').replace(/[*]\}/g, '+}')}`
                    integration = this.createPrivateIntegration(
                        gwName,
                        api,
                        this.vpcLink,
                        integrationUri,
                        route.verb,
                        route.execUnitName
                    )
                    this.execUnitToIntegration.set(route.execUnitName, integration)
                }
                apiRoutes.push(this.createRoute(gwName, api, '$default', integration))
                apiRoutes.push(this.createRoute(gwName, api, '$connect', integration))
                apiRoutes.push(this.createRoute(gwName, api, '$disconnect', integration))
            } else if (execUnit.type == 'lambda') {
                let integration: aws.apigatewayv2.Integration | undefined =
                    this.execUnitToIntegration.get(route.execUnitName)
                if (!integration) {
                    const lambda: aws.lambda.Function = this.lib.execUnitToFunctions.get(
                        route.execUnitName
                    )!
                    integration = this.createLambdaIntegration(
                        gwName,
                        api,
                        lambda.invokeArn,
                        this.lib.klothoVPC != undefined,
                        route.verb,
                        route.execUnitName
                    )
                    this.execUnitToIntegration.set(route.execUnitName, integration)
                }
                apiRoutes.push(this.createRoute(gwName, api, '$default', integration))
                apiRoutes.push(this.createRoute(gwName, api, '$connect', integration))
                apiRoutes.push(this.createRoute(gwName, api, '$disconnect', integration))
            } else if (execUnit.type == 'eks') {
                let integration: aws.apigatewayv2.Integration | undefined =
                    this.execUnitToIntegration.get(route.execUnitName)
                if (!integration) {
                    const nlb = this.lbPlugin.getExecUnitLoadBalancer(route.execUnitName)!
                    const integrationUri = pulumi.interpolate`http://${nlb.dnsName}${route.path
                        .replace(/:([^/]+)/g, '{$1}')
                        .replace(/[*]\}/g, '+}')}`
                    integration = this.createPrivateIntegration(
                        gwName,
                        api,
                        this.vpcLink,
                        integrationUri,
                        route.verb,
                        route.execUnitName
                    )
                    this.execUnitToIntegration.set(route.execUnitName, integration)
                }
                apiRoutes.push(this.createRoute(gwName, api, '$default', integration))
                apiRoutes.push(this.createRoute(gwName, api, '$connect', integration))
                apiRoutes.push(this.createRoute(gwName, api, '$disconnect', integration))
            } else {
                throw new Error('Unsuppotred integration time for api gateway')
            }
        }

        const deployment: aws.apigatewayv2.Deployment = this.createDeployment(
            gwName,
            api,
            apiRoutes
        )
        const stage: aws.apigatewayv2.Stage = this.createStage(
            gwName,
            api,
            deployment,
            this.vpcLink,
            'stage'
        )
        this.invokeUrls.push(stage.invokeUrl)
    }

    createDockerBasedAPIGateway(routes: Route[], providedName: string): void {
        const gwName = providedName.replace(/[^a-zA-Z0-9_-]/g, '-')
        const restAPI: aws.apigateway.RestApi = new aws.apigateway.RestApi(gwName, {
            binaryMediaTypes: ['application/octet-stream', 'image/*'],
        })
        const resourceMap = new Map<string, aws.apigateway.Resource>()
        const methods: aws.apigateway.Method[] = []
        const integrations: aws.apigateway.Integration[] = []
        const integrationNames: string[] = []
        const permissions: aws.lambda.Permission[] = []
        // create the resources and methods needed for the provided routes
        for (const r of routes) {
            const execUnit = this.lib.resourceIdToResource.get(`${r.execUnitName}_exec_unit`)
            const pathSegments = r.path.split('/').filter(Boolean)
            let methodPathLastPart = pathSegments.at(-1) ?? '/' // get the last part of the path
            let routeAndHash = `${methodPathLastPart.replace(':', '').replace('*', '')}-${sha256
                .sync(r.path)
                .slice(0, 5)}`
            // create the resources first
            // parent resource starts off null since we don't create the root resource
            let parentResource: aws.apigateway.Resource | null = null
            const methodRequestParams = {}
            const integrationRequestParams = {}
            let currPathSegments = ''
            for (let segment of pathSegments) {
                // Handle path parameters defined in express as :<param>
                if (segment.includes(':')) {
                    const pathParam = `request.path.${segment.replace(':', '').replace('*', '')}`
                    methodRequestParams[`method.${pathParam}`] = true
                    integrationRequestParams[`integration.${pathParam}`] = `method.${pathParam}`
                }

                segment = segment
                    .replace(/:([^/]+)/g, '{$1}') // convert express params :arg to AWS gateway {arg}
                    .replace(/[*]\}/g, '+}') // convert express greedy flag {arg*} to AWS gateway {arg+}
                    .replace(/\/\//g, '/') // collapse double '//' to single '/'
                currPathSegments += `${segment}/`
                if (resourceMap.has(currPathSegments)) {
                    parentResource = resourceMap.get(currPathSegments)!
                } else {
                    const resource = new aws.apigateway.Resource(
                        gwName + currPathSegments,
                        {
                            restApi: restAPI.id,
                            parentId:
                                parentResource == null ? restAPI.rootResourceId : parentResource.id,
                            pathPart: segment,
                        },
                        {
                            parent: restAPI,
                        }
                    )
                    resourceMap.set(currPathSegments, resource)
                    parentResource = resource
                }
            }

            //create the methods
            // We use the combination of the aws method property operationName alongside pulumi properties
            // replaceOnChanges and deleteBeforeReplace in order to correctly trigger swapping integrations
            // when infra is changed, for example from lambda to fargate. All three properties are required
            // to trigger a replace action of the method, which is required to correctly swap integrations
            // while preventing resource collisions on the method.
            const method = new aws.apigateway.Method(
                `${r.verb.toUpperCase()}-${routeAndHash}`,
                {
                    restApi: restAPI.id,
                    resourceId: parentResource?.id ?? restAPI.rootResourceId,
                    httpMethod: r.verb.toUpperCase(),
                    authorization: 'NONE',
                    operationName: `${execUnit.type}-${r.verb.toUpperCase()}-${routeAndHash}`,
                    requestParameters:
                        Object.keys(methodRequestParams).length == 0
                            ? undefined
                            : methodRequestParams,
                },
                {
                    parent: parentResource ?? restAPI,
                    replaceOnChanges: ['*'],
                    deleteBeforeReplace: true,
                }
            )
            methods.push(method)

            const integrationName = `${execUnit.type}-${r.verb.toUpperCase()}-${routeAndHash}`
            integrationNames.push(integrationName)
            if (execUnit.type == 'fargate') {
                const nlb = this.lib.execUnitToNlb.get(r.execUnitName)!
                const vpcLink = this.lib.execUnitToVpcLink.get(r.execUnitName)!
                integrations.push(
                    new aws.apigateway.Integration(
                        integrationName,
                        {
                            restApi: restAPI.id,
                            resourceId:
                                parentResource == null ? restAPI.rootResourceId : parentResource.id,
                            httpMethod: method.httpMethod,
                            integrationHttpMethod: method.httpMethod,
                            type: 'HTTP_PROXY',
                            connectionType: 'VPC_LINK',
                            connectionId: vpcLink.id,
                            uri: pulumi.interpolate`http://${nlb.loadBalancer.dnsName}${r.path
                                .replace(/:([^/]+)/g, '{$1}')
                                .replace(/[*]\}/g, '+}')}`,
                            requestParameters:
                                Object.keys(integrationRequestParams).length == 0
                                    ? undefined
                                    : integrationRequestParams,
                        },
                        {
                            parent: method,
                        }
                    )
                )
            } else if (execUnit.type == 'eks') {
                const nlb = this.lbPlugin.getExecUnitLoadBalancer(r.execUnitName)!
                const vpcLink = this.lib.execUnitToVpcLink.get(r.execUnitName)!
                integrations.push(
                    new aws.apigateway.Integration(
                        integrationName,
                        {
                            restApi: restAPI.id,
                            resourceId:
                                parentResource == null ? restAPI.rootResourceId : parentResource.id,
                            httpMethod: method.httpMethod,
                            integrationHttpMethod: method.httpMethod,
                            type: 'HTTP_PROXY',
                            connectionType: 'VPC_LINK',
                            connectionId: vpcLink.id,
                            uri: pulumi.interpolate`http://${nlb.dnsName}${r.path
                                .replace(/:([^/]+)/g, '{$1}')
                                .replace(/[*]\}/g, '+}')}`,
                            requestParameters:
                                Object.keys(integrationRequestParams).length == 0
                                    ? undefined
                                    : integrationRequestParams,
                        },
                        {
                            parent: method,
                        }
                    )
                )
            } else if (execUnit.type == 'lambda') {
                const lambda = this.lib.execUnitToFunctions.get(r.execUnitName)!
                integrations.push(
                    new aws.apigateway.Integration(
                        integrationName,
                        {
                            restApi: restAPI.id,
                            resourceId:
                                parentResource == null ? restAPI.rootResourceId : parentResource.id,
                            httpMethod: method.httpMethod,
                            integrationHttpMethod: 'POST',
                            type: 'AWS_PROXY',
                            uri: lambda.invokeArn,
                        },
                        {
                            parent: method,
                        }
                    )
                )

                const permissionName = `${r.verb}-${r.path.replace(/[^a-z0-9]/gi, '')}-permission`
                permissions.push(
                    new aws.lambda.Permission(permissionName, {
                        action: 'lambda:InvokeFunction',
                        function: lambda.name,
                        principal: 'apigateway.amazonaws.com',
                        sourceArn: pulumi.interpolate`arn:aws:execute-api:${this.lib.region}:${
                            this.lib.account.accountId
                        }:${restAPI.id}/*/${
                            r.verb.toUpperCase() === 'ANY' ? '*' : r.verb.toUpperCase()
                        }${parentResource == null ? '/' : parentResource.path}`,
                    })
                )
            }
        }

        // Create the deployment and stage
        const deployment = new aws.apigateway.Deployment(
            `${providedName}-deployment`,
            {
                restApi: restAPI,
                triggers: {
                    routes: sha256.sync(
                        routes
                            .map((r) => `${r.execUnitName}:${r.path}:${r.verb}`)
                            .sort()
                            .join()
                    ),
                    integrations: sha256.sync(
                        integrationNames
                            .map((i) => i)
                            .sort()
                            .join()
                    ),
                },
            },
            {
                dependsOn: [...methods, ...integrations, ...permissions],
                parent: restAPI,
            }
        )

        const stage = new aws.apigateway.Stage(
            `${providedName}-stage`,
            {
                deployment: deployment.id,
                restApi: restAPI.id,
                stageName: this.lib.stage,
            },
            {
                parent: deployment,
            }
        )

        this.lib.topologySpecOutputs.push(
            pulumi.all([restAPI.id, restAPI.urn]).apply(([id, urn]) => ({
                id: id,
                urn: urn,
                kind: '', // TODO
                type: 'AWS API Gateway',
                url: `https://console.aws.amazon.com/apigateway/home?region=${this.lib.region}#/apis/${id}/resources/`,
            }))
        )

        this.lib.gatewayToUrl.set(providedName, stage.invokeUrl)

        this.invokeUrls.push(stage.invokeUrl)
    }
}
