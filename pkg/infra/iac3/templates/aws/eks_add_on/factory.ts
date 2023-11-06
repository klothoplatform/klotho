import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AddonName: string
    ClusterName: pulumi.Input<string>
    Role: aws.iam.Role
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.eks.Addon {
    return new aws.eks.Addon(args.Name, {
        cluster: args.Cluster,
        addonName: args.AddOnName,
        //TMPL {{- if .Role }}
        serviceAccountRoleArn: args.Role.arn,
        //TMPL {{- end }}
    })
}
