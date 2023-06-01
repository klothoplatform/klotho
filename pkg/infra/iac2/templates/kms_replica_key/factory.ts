import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Description: string
    Enabled: boolean
    KeyPolicy: string
    PendingWindowInDays: number
    PrimaryKey: aws.kms.Key
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.kms.ReplicaKey {
    return new aws.kms.ReplicaKey(args.Name, {
        //TMPL {{- if .Description.Raw }}
        description: args.Description,
        //TMPL {{- end }}
        enabled: args.Enabled,
        policy: pulumi.jsonStringify(args.KeyPolicy),
        deletionWindowInDays: args.PendingWindowInDays,
        primaryKeyArn: args.PrimaryKey.arn,
    })
}
