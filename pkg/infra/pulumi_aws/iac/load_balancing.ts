import * as pulumi from '@pulumi/pulumi'
import * as aws from '@pulumi/aws'
import * as validators from './sanitization/aws/elb'
import {
    ListenerArgs,
    LoadBalancerArgs,
    TargetGroupArgs,
    TargetGroupAttachmentArgs,
} from '@pulumi/aws/lb'
import { ListenerRuleArgs } from '@pulumi/aws/alb'
import { hash as h, sanitized } from './sanitization/sanitizer'
import { CloudCCLib } from '../deploylib'

export interface Route {
    verb: string
    path: string
    execUnitName: string
}

export interface Gateway {
    Name: string
    Routes: Route[]
}

export class LoadBalancerPlugin {
    // A map of all resources which are going to be fronted by a load balancer
    private resourceIdToLB = new Map<string, aws.lb.LoadBalancer>()
    // A map of exec units to their target groups
    public readonly execUnitToTargetGroup: Map<string, aws.lb.TargetGroup> = new Map<
        string,
        aws.lb.TargetGroup
    >()

    public readonly execUnitToListner: Map<string, aws.lb.Listener> = new Map<
        string,
        aws.lb.Listener
    >()
    public readonly invokeUrls: pulumi.Output<string>[] = []

    constructor(private readonly lib: CloudCCLib) {}

    public createALBasGateways = (gateways: Gateway[]) => {
        gateways.forEach((gateway) => {
            this.invokeUrls.push(this.createALBasGateway(gateway))
        })
    }

    private subPathParametersForWildcard(route: string): string {
        const segments = route.split('/')
        const newRoute: string[] = []
        segments.forEach((seg) => {
            if (seg.startsWith(':')) {
                newRoute.push('*')
            } else {
                newRoute.push(seg)
            }
        })
        return newRoute.join('/')
    }

    private createALBasGateway = (gateway: Gateway): pulumi.Output<string> => {
        const albSG = new aws.ec2.SecurityGroup(`${this.lib.name}-${gateway.Name}`, {
            name: `${this.lib.name}-${gateway.Name}`,
            vpcId: this.lib.klothoVPC.id,
            egress: [
                {
                    cidrBlocks: ['0.0.0.0/0'],
                    description: 'Allows all outbound IPv4 traffic.',
                    fromPort: 0,
                    protocol: '-1',
                    toPort: 0,
                },
            ],
            ingress: [
                {
                    cidrBlocks: ['0.0.0.0/0'],
                    description: 'Allows all inbound IPv4 traffic.',
                    fromPort: 0,
                    protocol: '-1',
                    toPort: 0,
                },
            ],
        })

        const alb = this.createLoadBalancer(this.lib.name, gateway.Name, {
            internal: false,
            loadBalancerType: 'application',
            tags: { AppName: this.lib.name },
            subnets: this.lib.publicSubnetIds,
            securityGroups: [albSG.id],
        })

        for (const route of gateway.Routes) {
            let targetGroup = this.execUnitToTargetGroup.get(route.execUnitName)
            let listener = this.execUnitToListner.get(route.execUnitName)
            if (!targetGroup) {
                const execUnit = this.lib.resourceIdToResource.get(
                    `${route.execUnitName}_exec_unit`
                )
                if (['ecs', 'eks'].includes(execUnit.type)) {
                    targetGroup = this.createTargetGroup(this.lib.name, route.execUnitName, {
                        port: 3000,
                        protocol: 'HTTP',
                        vpcId: this.lib.klothoVPC.id,
                        targetType: 'ip',
                        tags: { AppName: this.lib.name },
                    })
                } else if (execUnit.type == 'lambda') {
                    targetGroup = this.createTargetGroup(this.lib.name, route.execUnitName, {
                        tags: { AppName: this.lib.name },
                    })
                } else {
                    throw new Error('unsupported execution unit type for ALB Gateway')
                }
            }
            if (!listener) {
                listener = this.createListener(this.lib.name, route.execUnitName, {
                    port: 80,
                    protocol: 'HTTP',
                    loadBalancerArn: alb.arn,
                    defaultActions: [
                        {
                            type: 'fixed-response',
                            fixedResponse: {
                                contentType: 'application/json',
                                statusCode: '404',
                            },
                        },
                    ],
                })
            }

            const listenConditions: pulumi.Unwrap<ListenerRuleArgs['conditions']> = [
                {
                    pathPattern: {
                        values: [
                            this.subPathParametersForWildcard(route.path),
                            `${this.subPathParametersForWildcard(route.path)}/`,
                        ],
                    },
                },
            ]
            if (route.verb.toUpperCase() != 'ANY') {
                listenConditions.push({
                    httpRequestMethod: {
                        values: [route.verb.toUpperCase()],
                    },
                })
            }
            this.createListenerRule(this.lib.name, route.execUnitName + route.path + route.verb, {
                listenerArn: listener!.arn,
                actions: [
                    {
                        type: 'forward',
                        targetGroupArn: targetGroup.arn,
                    },
                ],
                conditions: listenConditions,
            })
        }
        return alb.dnsName
    }

    public getExecUnitLoadBalancer = (execUnit: string): aws.lb.LoadBalancer | undefined => {
        return this.resourceIdToLB.get(execUnit)
    }

    public createLoadBalancer = (
        appName: string,
        resourceId: string,
        params: LoadBalancerArgs
    ): aws.lb.LoadBalancer => {
        let lb: aws.lb.LoadBalancer
        let lbName = sanitized(validators.loadBalancer.nameValidation())`${h(appName)}-${h(
            resourceId
        )}`
        switch (params.loadBalancerType) {
            case 'application':
                lb = new aws.lb.LoadBalancer(lbName, {
                    internal: params.internal || false,
                    loadBalancerType: 'application',
                    securityGroups: params.securityGroups,
                    subnets: params.subnets,
                    enableDeletionProtection: params.enableDeletionProtection || false,
                    tags: params.tags,
                })
                break
            case 'network':
                lb = new aws.lb.LoadBalancer(lbName, {
                    internal: params.internal || true,
                    loadBalancerType: 'network',
                    subnets: params.subnets,
                    enableDeletionProtection: params.enableDeletionProtection || false,
                    tags: params.tags,
                })
                break
            default:
                throw new Error('Unsupported load balancer type')
        }
        this.resourceIdToLB.set(resourceId, lb)
        return lb
    }

    public getExecUnitListener = (execUnit: string): aws.lb.Listener | undefined => {
        return this.execUnitToListner.get(execUnit)
    }

    public createListener = (
        appName: string,
        resourceId: string,
        params: ListenerArgs
    ): aws.lb.Listener => {
        const listener = new aws.lb.Listener(`${appName}-${resourceId}`, {
            loadBalancerArn: params.loadBalancerArn,
            defaultActions: params.defaultActions,
            port: params.port,
            protocol: params.protocol,
        })
        this.execUnitToListner.set(resourceId, listener)
        return listener
    }

    public createListenerRule = (
        appName: string,
        resourceId: string,
        params: ListenerRuleArgs
    ): aws.lb.ListenerRule => {
        return new aws.lb.ListenerRule(`${appName}-${resourceId}`, {
            listenerArn: params.listenerArn,
            actions: params.actions,
            conditions: params.conditions,
            priority: params.priority,
        })
    }

    public createTargetGroup = (
        appName: string,
        execUnitName: string,
        params: TargetGroupArgs
    ): aws.lb.TargetGroup => {
        let targetGroup: aws.lb.TargetGroup
        let tgName = sanitized(validators.targetGroup.nameValidation())`${h(appName)}-${h(
            execUnitName
        )}`
        if (params.targetType != 'lambda' && !(params.port && params.protocol)) {
            throw new Error('Port and Protocol must be specified for non lambda target types')
        }
        switch (params.targetType) {
            case 'ip':
                targetGroup = new aws.lb.TargetGroup(tgName, {
                    port: params.port,
                    protocol: params.protocol,
                    targetType: 'ip',
                    vpcId: params.vpcId,
                    tags: params.tags,
                })
                break
            case 'instance':
                targetGroup = new aws.lb.TargetGroup(tgName, {
                    port: params.port,
                    protocol: params.protocol,
                    vpcId: params.vpcId,
                    tags: params.tags,
                })
                break
            case 'alb':
                targetGroup = new aws.lb.TargetGroup(tgName, {
                    targetType: 'alb',
                    port: params.port,
                    protocol: params.protocol,
                    vpcId: params.vpcId,
                    loadBalancingAlgorithmType: params.loadBalancingAlgorithmType,
                    tags: params.tags,
                })
                break
            case 'lambda':
                targetGroup = new aws.lb.TargetGroup(tgName, {
                    targetType: 'lambda',
                    tags: params.tags,
                })
                break
            default:
                throw new Error('Unsupported target group target type')
        }
        this.execUnitToTargetGroup.set(execUnitName, targetGroup)
        return targetGroup
    }

    public attachTargetGroupToResource = (
        appName: string,
        resourceId: string,
        params: TargetGroupAttachmentArgs
    ): aws.lb.TargetGroupAttachment => {
        return new aws.lb.TargetGroupAttachment(`${appName}-${resourceId}`, {
            targetGroupArn: params.targetGroupArn,
            targetId: params.targetId,
            port: params.port,
        })
    }
}
