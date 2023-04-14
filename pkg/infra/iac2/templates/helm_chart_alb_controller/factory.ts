// A specialization of the more generic helm_chart/ template, for the ALB controller. This needs to create a service
// account, which itself needs things like VPC id and a role, so it's easier to just create it as a separate template.
import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    Namespace: string
    ClustersProvider: pulumi_k8s.Provider
    ClusterName: string
    Region: pulumi.Output<pulumi.UnwrappedObject<aws.GetRegionResult>>
    Vpc: aws.ec2.Vpc
    Role: aws.iam.Role
    dependsOn: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.helm.v3.Chart {
    return new pulumi_k8s.helm.v3.Chart(
        args.Name,
        {
            chart: 'https://aws.github.io/eks-charts',
            fetchOpts: {
                repo: 'https://aws.github.io/eks-charts',
            },
            version: '1.4.7',
            namespace: args.Namespace,
            values: {
                clusterName: args.ClusterName,
                serviceAccount: {
                    create: false,
                    name: new pulumi_k8s.core.v1.ServiceAccount(
                        `${args.ClusterName}-alb-controller`,
                        {
                            automountServiceAccountToken: true,
                            metadata: {
                                name: args.ClusterName + '-alb-controller',
                                namespace: args.Namespace,
                                annotations: {
                                    'eks.amazonaws.com/role-arn': pulumi.interpolate`${args.Role.arn}`,
                                },
                            },
                        }
                    ).metadata.name,
                },
                region: pulumi.interpolate`${args.Region.name}`,
                vpcId: pulumi.interpolate`${args.Vpc.id}`,
                podLabels: {
                    app: 'aws-lb-controller',
                },
            },
        },
        {
            provider: args.ClustersProvider,
            //TMPL {{- if .dependsOn.Raw }}
            dependsOn: args.dependsOn,
            //TMPL {{- end }}
        }
    )
}
