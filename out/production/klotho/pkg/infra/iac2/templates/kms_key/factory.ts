import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Description: string
    Enabled: boolean
    EnableKeyRotation: boolean
    KeyPolicy: string
    KeySpec: string
    KeyUsage: string
    MultiRegion: boolean
    PendingWindowInDays: number
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.kms.Key {
    return new aws.kms.Key(args.Name, {
        //TMPL {{- if .Description.Raw }}
        description: args.Description,
        //TMPL {{- end }}
        isEnabled: args.Enabled,
        enableKeyRotation: args.EnableKeyRotation,
        keyUsage: args.KeyUsage,
        customerMasterKeySpec: args.KeySpec,
        policy: pulumi.jsonStringify(args.KeyPolicy),
        deletionWindowInDays: args.PendingWindowInDays,
        multiRegion: args.MultiRegion,
    })
}
