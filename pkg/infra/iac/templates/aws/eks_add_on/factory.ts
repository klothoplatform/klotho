import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    AddonName: string
    ClusterName: pulumi.Input<string>
    Role: aws.iam.Role
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.eks.Addon {
    return new aws.eks.Addon(args.Name, {
        clusterName: args.Cluster.name,
        addonName: args.AddOnName,
        //TMPL {{- if .Role }}
        serviceAccountRoleArn: args.Role.arn,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
