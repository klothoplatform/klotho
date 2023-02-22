import * as aws from '@pulumi/aws'
import * as awsx from '@pulumi/awsx'
import * as pulumi from '@pulumi/pulumi'
import { ClusterArgs, FargateProfileArgs, GetClusterArgs } from '@pulumi/aws/eks'
import * as pulumi_k8s from '@pulumi/kubernetes'
import * as https from 'https'
import { getIssuerCAThumbprint } from '@pulumi/eks/cert-thumprint'
import path = require('path')
import * as certmanager from '@pulumi/kubernetes-cert-manager'

export const createTargetBinding = (
    execUnit: string,
    serviceName: pulumi.Output<string>,
    port: number,
    targetGroupArn: pulumi.Output<string>,
    provider: pulumi_k8s.Provider,
    parent,
    dependsOn
) => {
    return new pulumi_k8s.yaml.ConfigFile(
        `${execUnit}-${port}-tgb`,
        {
            file: './iac/k8s/add_ons/alb_controller/target_group_binding.yaml',
            transformations: [
                // Make every service private to the cluster, i.e., turn all services into ClusterIP instead of LoadBalancer.
                (obj: any, opts: pulumi.CustomResourceOptions) => {
                    obj.metadata = { name: `${execUnit}` }
                    obj.spec.serviceRef.name = serviceName
                    obj.spec.serviceRef.port = port
                    obj.spec.targetGroupARN = targetGroupArn
                },
            ],
        },
        { provider, parent, dependsOn }
    )
}

export const attachPermissionsToRole = (role: aws.iam.Role): void => {
    const ingressControllerPolicy = new aws.iam.Policy('ingressController-iam-policy', {
        policy: {
            Version: '2012-10-17',
            Statement: [
                {
                    Effect: 'Allow',
                    Action: ['iam:CreateServiceLinkedRole'],
                    Resource: '*',
                    Condition: {
                        StringEquals: {
                            'iam:AWSServiceName': 'elasticloadbalancing.amazonaws.com',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'ec2:DescribeAccountAttributes',
                        'ec2:DescribeAddresses',
                        'ec2:DescribeAvailabilityZones',
                        'ec2:DescribeInternetGateways',
                        'ec2:DescribeVpcs',
                        'ec2:DescribeVpcPeeringConnections',
                        'ec2:DescribeSubnets',
                        'ec2:DescribeSecurityGroups',
                        'ec2:DescribeInstances',
                        'ec2:DescribeNetworkInterfaces',
                        'ec2:DescribeTags',
                        'ec2:GetCoipPoolUsage',
                        'ec2:DescribeCoipPools',
                        'elasticloadbalancing:DescribeLoadBalancers',
                        'elasticloadbalancing:DescribeLoadBalancerAttributes',
                        'elasticloadbalancing:DescribeListeners',
                        'elasticloadbalancing:DescribeListenerCertificates',
                        'elasticloadbalancing:DescribeSSLPolicies',
                        'elasticloadbalancing:DescribeRules',
                        'elasticloadbalancing:DescribeTargetGroups',
                        'elasticloadbalancing:DescribeTargetGroupAttributes',
                        'elasticloadbalancing:DescribeTargetHealth',
                        'elasticloadbalancing:DescribeTags',
                    ],
                    Resource: '*',
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'cognito-idp:DescribeUserPoolClient',
                        'acm:ListCertificates',
                        'acm:DescribeCertificate',
                        'iam:ListServerCertificates',
                        'iam:GetServerCertificate',
                        'waf-regional:GetWebACL',
                        'waf-regional:GetWebACLForResource',
                        'waf-regional:AssociateWebACL',
                        'waf-regional:DisassociateWebACL',
                        'wafv2:GetWebACL',
                        'wafv2:GetWebACLForResource',
                        'wafv2:AssociateWebACL',
                        'wafv2:DisassociateWebACL',
                        'shield:GetSubscriptionState',
                        'shield:DescribeProtection',
                        'shield:CreateProtection',
                        'shield:DeleteProtection',
                    ],
                    Resource: '*',
                },
                {
                    Effect: 'Allow',
                    Action: ['ec2:AuthorizeSecurityGroupIngress', 'ec2:RevokeSecurityGroupIngress'],
                    Resource: '*',
                },
                {
                    Effect: 'Allow',
                    Action: ['ec2:CreateSecurityGroup'],
                    Resource: '*',
                },
                {
                    Effect: 'Allow',
                    Action: ['ec2:CreateTags'],
                    Resource: 'arn:aws:ec2:*:*:security-group/*',
                    Condition: {
                        StringEquals: {
                            'ec2:CreateAction': 'CreateSecurityGroup',
                        },
                        Null: {
                            'aws:RequestTag/elbv2.k8s.aws/cluster': 'false',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: ['ec2:CreateTags', 'ec2:DeleteTags'],
                    Resource: 'arn:aws:ec2:*:*:security-group/*',
                    Condition: {
                        Null: {
                            'aws:RequestTag/elbv2.k8s.aws/cluster': 'true',
                            'aws:ResourceTag/elbv2.k8s.aws/cluster': 'false',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'ec2:AuthorizeSecurityGroupIngress',
                        'ec2:RevokeSecurityGroupIngress',
                        'ec2:DeleteSecurityGroup',
                    ],
                    Resource: '*',
                    Condition: {
                        Null: {
                            'aws:ResourceTag/elbv2.k8s.aws/cluster': 'false',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'elasticloadbalancing:CreateLoadBalancer',
                        'elasticloadbalancing:CreateTargetGroup',
                    ],
                    Resource: '*',
                    Condition: {
                        Null: {
                            'aws:RequestTag/elbv2.k8s.aws/cluster': 'false',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'elasticloadbalancing:CreateListener',
                        'elasticloadbalancing:DeleteListener',
                        'elasticloadbalancing:CreateRule',
                        'elasticloadbalancing:DeleteRule',
                    ],
                    Resource: '*',
                },
                {
                    Effect: 'Allow',
                    Action: ['elasticloadbalancing:AddTags', 'elasticloadbalancing:RemoveTags'],
                    Resource: [
                        'arn:aws:elasticloadbalancing:*:*:targetgroup/*/*',
                        'arn:aws:elasticloadbalancing:*:*:loadbalancer/net/*/*',
                        'arn:aws:elasticloadbalancing:*:*:loadbalancer/app/*/*',
                    ],
                    Condition: {
                        Null: {
                            'aws:RequestTag/elbv2.k8s.aws/cluster': 'true',
                            'aws:ResourceTag/elbv2.k8s.aws/cluster': 'false',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: ['elasticloadbalancing:AddTags', 'elasticloadbalancing:RemoveTags'],
                    Resource: [
                        'arn:aws:elasticloadbalancing:*:*:listener/net/*/*/*',
                        'arn:aws:elasticloadbalancing:*:*:listener/app/*/*/*',
                        'arn:aws:elasticloadbalancing:*:*:listener-rule/net/*/*/*',
                        'arn:aws:elasticloadbalancing:*:*:listener-rule/app/*/*/*',
                    ],
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'elasticloadbalancing:ModifyLoadBalancerAttributes',
                        'elasticloadbalancing:SetIpAddressType',
                        'elasticloadbalancing:SetSecurityGroups',
                        'elasticloadbalancing:SetSubnets',
                        'elasticloadbalancing:DeleteLoadBalancer',
                        'elasticloadbalancing:ModifyTargetGroup',
                        'elasticloadbalancing:ModifyTargetGroupAttributes',
                        'elasticloadbalancing:DeleteTargetGroup',
                    ],
                    Resource: '*',
                    Condition: {
                        Null: {
                            'aws:ResourceTag/elbv2.k8s.aws/cluster': 'false',
                        },
                    },
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'elasticloadbalancing:RegisterTargets',
                        'elasticloadbalancing:DeregisterTargets',
                    ],
                    Resource: 'arn:aws:elasticloadbalancing:*:*:targetgroup/*/*',
                },
                {
                    Effect: 'Allow',
                    Action: [
                        'elasticloadbalancing:SetWebAcl',
                        'elasticloadbalancing:ModifyListener',
                        'elasticloadbalancing:AddListenerCertificates',
                        'elasticloadbalancing:RemoveListenerCertificates',
                        'elasticloadbalancing:ModifyRule',
                    ],
                    Resource: '*',
                },
            ],
        },
    })

    new aws.iam.RolePolicyAttachment('eks-albRole-policy-attach', {
        policyArn: ingressControllerPolicy.arn,
        role: role,
    })

    return
}

export const installLoadBalancerController = (
    clusterName: pulumi.Output<string> | string,
    namespace: string,
    serviceAccount: pulumi_k8s.core.v1.ServiceAccount,
    vpc: awsx.ec2.Vpc,
    provider: pulumi_k8s.Provider,
    region: string,
    fargate: boolean,
    dependsOn?
): pulumi_k8s.helm.v3.Chart => {
    /**
     * we can't limit the permissions to a specific resource because the load balancer controller doesn't know the names of the resources or arns that it is creating.
     *  At best what we can do is prefix every k8s service and add a wildcard to the resource,
     *  but then if someone used the cluster controller to create LBs outside of klotho the permissions would get denied
     */
    const tranfsformation = (obj, opts): void => {
        if (obj.kind == 'CustomResourceDefinition') {
            delete obj.status
        }
        return
    }

    // Declare the ALBIngressController in 1 step with the Helm Chart.
    return new pulumi_k8s.helm.v3.Chart(
        `${clusterName}-alb-c`,
        {
            chart: 'aws-load-balancer-controller',
            fetchOpts: { repo: 'https://aws.github.io/eks-charts' },
            values: {
                clusterName,
                serviceAccount: {
                    create: false,
                    name: serviceAccount.metadata.name,
                },
                region,
                vpcId: vpc.id,
                podLabels: {
                    app: 'aws-lb-controller',
                },
                enableCertManager: !fargate,
            },
            version: '1.4.7',
            namespace: namespace,
            transformations: [tranfsformation],
        },
        { provider, dependsOn }
    )
}
