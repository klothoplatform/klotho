import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecr.Repository {
    return new aws.ecr.Repository(args.Name, {
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
