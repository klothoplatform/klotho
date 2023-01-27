import { Region } from '@pulumi/aws'
import * as aws from '@pulumi/aws'
import * as awsx from '@pulumi/awsx'
import * as k8s from '@pulumi/kubernetes'

import * as pulumi from '@pulumi/pulumi'
import * as sha256 from 'simple-sha256'
import * as fs from 'fs'
import * as requestRetry from 'requestretry'
import * as crypto from 'crypto'

import * as eks from '@pulumi/eks'
import {
    ListenerArgs,
    TargetGroupArgs,
    LoadBalancerArgs,
    TargetGroupAttachmentArgs,
} from '@pulumi/aws/lb'
import { ListenerRuleArgs } from '@pulumi/aws/alb'
import { CloudCCLib } from '../deploylib'

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
                if (['fargate', 'eks'].includes(execUnit.type)) {
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
                                statusCode: '4XX',
                            },
                        },
                    ],
                })
            }

            this.createListenerRule(this.lib.name, route.execUnitName + route.path, {
                listenerArn: listener!.arn,
                actions: [
                    {
                        type: 'forward',
                        targetGroupArn: targetGroup.arn,
                    },
                ],
                conditions: [
                    {
                        pathPattern: {
                            values: [route.path],
                        },
                    },
                ],
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
        switch (params.loadBalancerType) {
            case 'application':
                lb = new aws.lb.LoadBalancer(`${appName}-${resourceId}`, {
                    internal: params.internal || false,
                    loadBalancerType: 'application',
                    securityGroups: params.securityGroups,
                    subnets: params.subnets,
                    enableDeletionProtection: params.enableDeletionProtection || false,
                    tags: params.tags,
                })
                break
            case 'network':
                lb = new aws.lb.LoadBalancer(`${appName}-${resourceId}`, {
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
        if (params.targetType != 'lambda' && !(params.port && params.protocol)) {
            throw new Error('Port and Protocol must be specified for non lambda target types')
        }
        switch (params.targetType) {
            case 'ip':
                targetGroup = new aws.lb.TargetGroup(`${appName}-${execUnitName}`, {
                    port: params.port,
                    protocol: params.protocol,
                    targetType: 'ip',
                    vpcId: params.vpcId,
                    tags: params.tags,
                })
                break
            case 'instance':
                targetGroup = new aws.lb.TargetGroup(`${appName}-${execUnitName}`, {
                    port: params.port,
                    protocol: params.protocol,
                    vpcId: params.vpcId,
                    tags: params.tags,
                })
                break
            case 'alb':
                targetGroup = new aws.lb.TargetGroup(`${appName}-${execUnitName}`, {
                    targetType: 'alb',
                    port: params.port,
                    protocol: params.protocol,
                    vpcId: params.vpcId,
                    loadBalancingAlgorithmType: params.loadBalancingAlgorithmType,
                    tags: params.tags,
                })
                break
            case 'lambda':
                targetGroup = new aws.lb.TargetGroup(`${appName}-${execUnitName}`, {
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
