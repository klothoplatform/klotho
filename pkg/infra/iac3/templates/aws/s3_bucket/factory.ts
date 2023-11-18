import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    ForceDestroy: boolean
    IndexDocument: string
    SSEAlgorithm: string
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.s3.Bucket {
    return new aws.s3.Bucket(
        args.Name,
        {
            forceDestroy: args.ForceDestroy,
            //TMPL {{- if .SSEAlgorithm }}
            serverSideEncryptionConfiguration: {
                rule: {
                    applyServerSideEncryptionByDefault: {
                        sseAlgorithm: args.SSEAlgorithm,
                    },
                    bucketKeyEnabled: true,
                },
            },
            //TMPL {{- end }}
            //TMPL {{- if .IndexDocument }}
            website: {
                indexDocument: args.IndexDocument,
            },
            //TMPL {{- end }}
        },
        { protect: args.protect }
    )
}
function properties(object: aws.s3.Bucket, args: Args) {
    return {
        AllBucketDirectory: pulumi.interpolate`${object.arn}/*`,
        Arn: object.arn,
        BucketRegionalDomainName: object.bucketRegionalDomainName,
        BucketName: object.bucket,
    }
}
