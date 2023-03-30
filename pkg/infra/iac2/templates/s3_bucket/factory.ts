import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    ForceDestroy: boolean
    IndexDocument: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.s3.Bucket {
    return new aws.s3.Bucket(
        args.Name,
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
            website: {
                indexDocument: args.IndexDocument,
            },
        },
        { protect: true }
    )
}
