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
        clusterName: args.ClusterName,
        addonName: args.AddonName,
        //TMPL {{- if .Role.Raw }}
        serviceAccountRoleArn: args.Role.arn,
        //TMPL {{- end }}
    })
}
