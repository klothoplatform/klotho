import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    SanitizedName: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecr.Repository {
    return new aws.ecr.Repository(args.SanitizedName, {
        imageScanningConfiguration: {
            scanOnPush: true,
        },
        imageTagMutability: 'MUTABLE',
        forceDelete: true,
        encryptionConfigurations: [{ encryptionType: 'KMS' }],
        tags: {
            env: 'production',
            AppName: args.Name,
        },
    })
}
