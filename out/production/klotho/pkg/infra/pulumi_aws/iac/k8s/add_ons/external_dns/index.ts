import * as aws from '@pulumi/aws'
import * as pulumi_k8s from '@pulumi/kubernetes'

export const installExternalDNS = (
    namespace: string,
    provider: pulumi_k8s.Provider,
    serviceAccount: pulumi_k8s.core.v1.ServiceAccount,
    dependsOn?
): pulumi_k8s.helm.v3.Chart => {
    // Declare the ALBIngressController in 1 step with the Helm Chart.
    return new pulumi_k8s.helm.v3.Chart(
        'external-dns',
        {
            chart: 'external-dns',
            fetchOpts: { repo: 'https://kubernetes-sigs.github.io/external-dns/' },
            values: {
                extraArgs: {
                    'aws-sd-service-cleanup': true,
                },
                serviceAccount: {
                    create: false,
                    name: serviceAccount.metadata.name,
                },
                provider: 'aws-sd',
                'aws-sd-service-cleanup': true,
            },
            namespace: namespace,
        },
        { provider, dependsOn }
    )
}

export const attachPermissionsToRole = (role: aws.iam.Role) => {
    const ingressControllerPolicy = new aws.iam.Policy('externalDNS-iam-policy', {
        policy: {
            Version: '2012-10-17',
            Statement: [
                {
                    Effect: 'Allow',
                    Action: [
                        'route53:GetHostedZone',
                        'route53:ListHostedZonesByName',
                        'route53:CreateHostedZone',
                        'route53:DeleteHostedZone',
                        'route53:ChangeResourceRecordSets',
                        'route53:CreateHealthCheck',
                        'route53:GetHealthCheck',
                        'route53:DeleteHealthCheck',
                        'route53:UpdateHealthCheck',
                        'ec2:DescribeVpcs',
                        'ec2:DescribeRegions',
                        'servicediscovery:*',
                    ],
                    Resource: ['*'],
                },
            ],
        },
    })

    new aws.iam.RolePolicyAttachment('eks-externalDns-policy-attach', {
        policyArn: ingressControllerPolicy.arn,
        role: role,
    })
}
