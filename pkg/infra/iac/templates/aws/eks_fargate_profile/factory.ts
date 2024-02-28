import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { TemplateWrapper, ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    Cluster: aws.eks.Cluster
    PodExecutionRole: aws.iam.Role
    Selectors: TemplateWrapper<
        pulumi.Input<pulumi.Input<aws.types.input.eks.FargateProfileSelector>[]>
    >
    Tags: ModelCaseWrapper<Record<string, string>>
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
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        {
            customTimeouts: { create: '30m', update: '30m', delete: '30m' },
        }
    )
}
