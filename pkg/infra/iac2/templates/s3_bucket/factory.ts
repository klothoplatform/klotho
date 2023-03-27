import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AccountId: aws.GetCallerIdentityResult
    ForceDestroy: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.s3.Bucket {
    return new aws.s3.Bucket(
        `${args.AccountId}-${args.Name}`,
        {
            forceDestroy: args.ForceDestroy,
            serverSideEncryptionConfiguration: {
                rule: {
                    applyServerSideEncryptionByDefault: {
                        sseAlgorithm: 'aws:kms',
                    },
                    bucketKeyEnabled: true,
                },
            },
        },
        { protect: true }
    )
}
