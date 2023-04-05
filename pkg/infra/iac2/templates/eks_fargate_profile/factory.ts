import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as aws_native from '@pulumi/aws-native'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    Cluster: aws_native.eks.Cluster
    PodExecutionRole: aws.iam.Role
    Selectors: pulumi.Input<pulumi.Input<aws.types.input.eks.FargateProfileSelector>[]>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.eks.FargateProfile {
    return new aws.eks.FargateProfile(
        args.Name,
        {
            clusterName: args.Cluster.name,
            podExecutionRoleArn: args.PodExecutionRole.arn,
            selectors: args.Selectors,
            subnetIds: args.Subnets.map((subnet) => subnet.id),
        },
        {
            customTimeouts: { create: '30m', update: '30m', delete: '30m' },
        }
    )
}
