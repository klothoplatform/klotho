import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    AccountId: pulumi.Output<pulumi.UnwrappedObject<aws.GetCallerIdentityResult>>
    ForceDestroy: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi.Output<aws.s3.Bucket> {
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
