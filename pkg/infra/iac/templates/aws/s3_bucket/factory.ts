import * as pulumi from '@pulumi/pulumi'
import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    ForceDestroy: boolean
    IndexDocument: string
    SSEAlgorithm: string
    protect: boolean
    Tags: ModelCaseWrapper<Record<string, string>>
    Bucket: string
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
            //TMPL {{- if .Tags }}
            tags: args.Tags,
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
        Bucket: object.bucket,
        // DS - Leaving BucketName in place for backward compatibility for fow
        BucketName: object.bucket,
    }
}

function infraExports(
    object: ReturnType<typeof create>,
    args: Args,
    props: ReturnType<typeof properties>
) {
    return {
        BucketName: object.bucket,
    }
}

function importResource(args: Args): aws.s3.Bucket {
    return aws.s3.Bucket.get(args.Name, args.Bucket)
}
