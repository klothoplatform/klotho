import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { CloudCCLib } from '../deploylib'
import * as sha256 from 'simple-sha256'
import { LoadBalancerPlugin } from './load_balancing'
import { DeploymentArgs, StageArgs } from '@pulumi/aws/apigatewayv2'

export interface Route {
    verb: string
    path: string
    execUnitName: string
}

export interface Gateway {
    Name: string
    Routes: Route[]
    ApiType: 'REST' | 'HTTP'
}

function sanitizeName(g: Gateway) {
    return g.Name.replace(/[^a-zA-Z0-9_-]+/g, '-')
}

export class ApiGateway {
    private readonly vpcLink?: aws.apigatewayv2.VpcLink
    public readonly invokeUrls: pulumi.Output<string>[] = []
    private readonly execUnitToIntegration = new Map<string, aws.apigatewayv2.Integration>()

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
            switch (gateway.ApiType) {
                case 'REST':
                    this.createRestGateway(gateway)
                    break
                case 'HTTP':
                    this.createHttpGateway(gateway)
                    break
            }
        })
    }

    get appName() {
        return this.lib.name
    }

    get accountId() {
        return this.lib.account.accountId
    }

    createVpcLink(
        securityGroupIds: pulumi.Output<string>[],
        subnetIds: pulumi.Output<string[]>
    ): aws.apigatewayv2.VpcLink {
        return new aws.apigatewayv2.VpcLink(`${this.appName}`, {
            securityGroupIds,
            subnetIds,
        })
    }

    createLambdaPermission(gwName: string, api: aws.apigatewayv2.Api, r: Route) {
        const verb = r.verb.toUpperCase()
        const path = this.convertPath(r.path)
        const resourceId = pulumi.interpolate`${api.id}/*/${verb == 'ANY' ? '*' : verb}${
            path == '' ? '/' : path
        }`
        const lambda = this.lib.execUnitToFunctions.get(r.execUnitName)!
        return new aws.lambda.Permission(
            `${gwName}-${verb}-${path}`,
            {
                statementId: `${gwName}-http-${verb}-${r.path.replace(/[^a-zA-Z0-9]+/g, '-')}`,
                action: 'lambda:InvokeFunction',
                function: lambda,
                principal: 'apigateway.amazonaws.com',
                sourceArn: pulumi.interpolate`arn:aws:execute-api:${this.lib.region}:${this.accountId}:${resourceId}`,
            },
            {
                dependsOn: [api],
                parent: lambda,
            }
        )
    }

    integrationName(r: Route) {
        const execUnit = this.lib.resourceIdToResource.get(`${r.execUnitName}_exec_unit`)
        const verb = r.verb.toUpperCase()
        const path = this.convertPath(r.path)
        return `${verb}-${path}-${execUnit.type}`
    }

    createHTTPIntegration(api: aws.apigatewayv2.Api, r: Route): aws.apigatewayv2.Integration {
        const execUnit = this.lib.resourceIdToResource.get(`${r.execUnitName}_exec_unit`)
        const integrationName = this.integrationName(r)
        switch (execUnit.type) {
            case 'ecs':
                const ecsNlb = this.lib.execUnitToNlb.get(r.execUnitName)!
                return new aws.apigatewayv2.Integration(
                    integrationName,
                    {
                        apiId: api.id,
                        integrationType: 'HTTP_PROXY',
                        integrationMethod: 'ANY',
                        integrationUri: ecsNlb.loadBalancer.arn,
                        connectionType: 'VPC_LINK',
                        connectionId: '${stageVariables.vpcLinkId}',
                    },
                    {
                        parent: api,
                    }
                )
            case 'eks':
                const eksListener = this.lbPlugin.getExecUnitListener(r.execUnitName)!
                return new aws.apigatewayv2.Integration(
                    integrationName,
                    {
                        apiId: api.id,
                        integrationType: 'HTTP_PROXY',
                        integrationMethod: 'ANY',
                        integrationUri: eksListener.arn,
                        connectionType: 'VPC_LINK',
                        connectionId: '${stageVariables.vpcLinkId}',
                    },
                    {
                        parent: api,
                    }
                )
            case 'lambda':
                const lambda = this.lib.execUnitToFunctions.get(r.execUnitName)!
                return new aws.apigatewayv2.Integration(
                    integrationName,
                    {
                        apiId: api.id,
                        integrationType: 'AWS_PROXY',
                        integrationMethod: 'POST',
                        integrationUri: lambda.arn,
                        payloadFormatVersion: '2.0',
                    },
                    {
                        parent: api,
                    }
                )
            default:
                throw new Error(`Unsupported execution unit type: ${execUnit.type}`)
        }
    }

    createRoute(
        api: aws.apigatewayv2.Api,
        routeKey: string,
        integration: aws.apigatewayv2.Integration
    ): aws.apigatewayv2.Route {
        return new aws.apigatewayv2.Route(
            routeKey,
            {
                apiId: api.id,
                routeKey,
                target: pulumi.interpolate`integrations/${integration.id}`,
            },
            {
                parent: integration,
            }
        )
    }

    createDeployment(
        gwName: string,
        api: aws.apigatewayv2.Api,
        integrationNames: string[],
        routes: aws.apigatewayv2.Route[],
        dependsOn: pulumi.Resource[] = []
    ) {
        const triggers: DeploymentArgs['triggers'] = {
            integrationNames: sha256.sync(integrationNames.sort().join(',')),
        }
        if (this.vpcLink != undefined) {
            triggers.link = this.vpcLink.arn
        }
        return new aws.apigatewayv2.Deployment(
            `${gwName}-deploy`,
            {
                apiId: api.id,
                triggers,
            },
            {
                dependsOn: [...routes, ...dependsOn],
                parent: api,
            }
        )
    }

    createStage(
        gwName: string,
        api: aws.apigatewayv2.Api,
        deployment: aws.apigatewayv2.Deployment
    ) {
        const dependsOn: pulumi.Resource[] = []
        const stageVariables: StageArgs['stageVariables'] = {}
        if (this.vpcLink != undefined) {
            stageVariables.vpcLinkId = this.vpcLink.id
            dependsOn.push(this.vpcLink)
        }
        return new aws.apigatewayv2.Stage(
            `${gwName}-stage`,
            {
                apiId: api.id,
                name: '$default',
                deploymentId: deployment.id,
                stageVariables,
            },
            {
                dependsOn,
                parent: api,
            }
        )
    }

    createWebSocketGateway(gateway: Gateway) {
        const gwName = sanitizeName(gateway)
        const units = new Set<string>()
        gateway.Routes.forEach((gw) => units.add(gw.execUnitName))
        if (units.size > 1) {
            throw new Error(`only one exec unit is supported for websocket API Gateway ${gwName}`)
        }

        const api: aws.apigatewayv2.Api = new aws.apigatewayv2.Api(gwName, {
            name: `${this.appName}-${gwName}`,
            protocolType: 'WEBSOCKET',
            // routeSelectionExpression currently not used, the gateway only uses
            // the builtin routes, $default, $connect, and $disconnect
            routeSelectionExpression: `$request.body.action`,
        })
        const apiRoutes: aws.apigatewayv2.Route[] = []
        const lambdaPermissions: aws.lambda.Permission[] = []
        const integrationNames: string[] = []

        for (const route of gateway.Routes) {
            const integration = this.createHTTPIntegration(api, route)

            this.execUnitToIntegration.set(route.execUnitName, integration)
            apiRoutes.push(this.createRoute(api, '$default', integration))
            apiRoutes.push(this.createRoute(api, '$connect', integration))
            apiRoutes.push(this.createRoute(api, '$disconnect', integration))

            const execUnit = this.lib.resourceIdToResource.get(`${route.execUnitName}_exec_unit`)
            if (execUnit.type == 'lambda') {
                lambdaPermissions.push(this.createLambdaPermission(gwName, api, route))
            }

            const integrationName = this.integrationName(route)
            integrationNames.push(integrationName)
        }

        const deployment: aws.apigatewayv2.Deployment = this.createDeployment(
            gwName,
            api,
            integrationNames,
            apiRoutes,
            lambdaPermissions
        )
        const stage: aws.apigatewayv2.Stage = this.createStage(gwName, api, deployment)
        this.invokeUrls.push(stage.invokeUrl)
    }

    createHttpGateway(gateway: Gateway) {
        const gwName = sanitizeName(gateway)
        const api: aws.apigatewayv2.Api = new aws.apigatewayv2.Api(`${this.appName}-${gwName}`, {
            name: `${this.appName}-${gwName}`,
            protocolType: 'HTTP',
            routeSelectionExpression: '$request.method $request.path',
            tags: {
                'Klotho:app': this.appName,
            },
        })

        const apiRoutes: aws.apigatewayv2.Route[] = []
        const lambdaPermissions: aws.lambda.Permission[] = []
        const integrationNames: string[] = []

        for (const r of gateway.Routes) {
            const integration = this.createHTTPIntegration(api, r)

            // routeKey match against api.routeSelectionExpression
            let routeKey = `${r.verb.toUpperCase()} ${this.convertPath(r.path)}`
            if (routeKey == 'ANY /') {
                // For catch-all, use idiomatic $default instead
                routeKey = '$default'
            }
            const route = this.createRoute(api, routeKey, integration)
            apiRoutes.push(route)

            const execUnit = this.lib.resourceIdToResource.get(`${r.execUnitName}_exec_unit`)
            if (execUnit.type == 'lambda') {
                lambdaPermissions.push(this.createLambdaPermission(gwName, api, r))
            }

            const integrationName = this.integrationName(r)
            integrationNames.push(integrationName)
        }

        const deploy = this.createDeployment(
            gwName,
            api,
            integrationNames,
            apiRoutes,
            lambdaPermissions
        )

        const stage = this.createStage(gwName, api, deploy)

        this.lib.gatewayToUrl.set(gateway.Name, stage.invokeUrl)
        this.invokeUrls.push(stage.invokeUrl)
    }

    /**
     * Converts an express style path (or path segment) to API Gateway compatible.
     */
    convertPath(path: string): string {
        return path
            .replace(/:([^/]+)/g, '{$1}') // convert express params :arg to AWS gateway {arg}
            .replace(/[*]\}/g, '+}') // convert express greedy flag {arg*} to AWS gateway {arg+}
            .replace(/\/\//g, '/') // collapse double '//' to single '/'
    }

    createRestGateway(gateway: Gateway): void {
        const gwName = sanitizeName(gateway)
        const restAPI: aws.apigateway.RestApi = new aws.apigateway.RestApi(gwName, {
            binaryMediaTypes: ['application/octet-stream', 'image/*'],
        })
        const resourceMap = new Map<string, aws.apigateway.Resource>()
        const methods: aws.apigateway.Method[] = []
        const integrations: aws.apigateway.Integration[] = []
        const integrationNames: string[] = []
        const permissions: aws.lambda.Permission[] = []
        // create the resources and methods needed for the provided routes
        for (const r of gateway.Routes) {
            const execUnit = this.lib.resourceIdToResource.get(`${r.execUnitName}_exec_unit`)
            const pathSegments = r.path.split('/').filter(Boolean)
            let methodPathLastPart = pathSegments[pathSegments.length - 1] ?? '/' // get the last part of the path
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

                segment = this.convertPath(segment)

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
            if (execUnit.type == 'ecs') {
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
                            uri: pulumi.interpolate`http://${
                                nlb.loadBalancer.dnsName
                            }${this.convertPath(r.path)}`,
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
                            uri: pulumi.interpolate`http://${nlb.dnsName}${this.convertPath(
                                r.path
                            )}`,
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
                const integration = new aws.apigateway.Integration(
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
                integrations.push(integration)

                permissions.push(
                    new aws.lambda.Permission(
                        `${r.execUnitName}-${r.verb}-${r.path}-permission`,
                        {
                            statementId: `${gwName}-rest-${r.verb}-${r.path.replace(
                                /[^a-zA-Z0-9]/g,
                                '-'
                            )}`,
                            action: 'lambda:InvokeFunction',
                            function: lambda.name,
                            principal: 'apigateway.amazonaws.com',
                            sourceArn: pulumi.interpolate`arn:aws:execute-api:${this.lib.region}:${
                                this.accountId
                            }:${restAPI.id}/*/${
                                r.verb.toUpperCase() === 'ANY' ? '*' : r.verb.toUpperCase()
                            }${parentResource == null ? '/' : parentResource.path}`,
                        },
                        {
                            dependsOn: [restAPI],
                            parent: lambda,
                        }
                    )
                )
            }
        }

        // Create the deployment and stage
        const deployment = new aws.apigateway.Deployment(
            `${gwName}-deployment`,
            {
                restApi: restAPI,
                triggers: {
                    routes: sha256.sync(
                        gateway.Routes.map((r) => `${r.execUnitName}:${r.path}:${r.verb}`)
                            .sort()
                            .join()
                    ),
                    integrations: sha256.sync(
                        integrationNames
                            .map((i) => i)
                            .sort()
                            .join()
                    ),
                    connections: pulumi.all(integrations.map((i) => i.connectionId)).apply((is) =>
                        sha256.sync(
                            is
                                .filter((i) => i)
                                .sort()
                                .join()
                        )
                    ),
                },
            },
            {
                dependsOn: [...methods, ...integrations, ...permissions],
                parent: restAPI,
            }
        )

        const stage = new aws.apigateway.Stage(
            `${gwName}-stage`,
            {
                deployment: deployment.id,
                restApi: restAPI.id,
                // TODO update this to '$default' so the stage isn't part of the invoke URL
                // https://github.com/klothoplatform/klotho/issues/235
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

        this.lib.gatewayToUrl.set(gateway.Name, stage.invokeUrl)
        this.invokeUrls.push(stage.invokeUrl)
    }
}
