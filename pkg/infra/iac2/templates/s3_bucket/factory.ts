import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    ForceDestroy: boolean
    IndexDocument: string
    protect: boolean
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
            //TMPL {{ if .IndexDocument.Raw }}
            website: {
                indexDocument: args.IndexDocument,
            },
            //TMPL {{ end }}
        },
        { protect: args.protect }
    )
}
