import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AccountId: pulumi.Output<string>
    ForceDestroy: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Role {
    return args.AccountId.apply((accountId) => {
        return new aws.s3.Bucket(
            `${accountId}-${args.Name}`,
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
    })
}
