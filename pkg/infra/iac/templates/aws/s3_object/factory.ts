import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as mime from 'mime'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Bucket: aws.s3.Bucket
    Key: string
    FilePath: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.s3.BucketObject {
    return new aws.s3.BucketObject(args.Name, {
        bucket: args.Bucket,
        key: args.Key,
        source: new pulumi.asset.FileAsset(args.FilePath), // use FileAsset to point to a file
        contentType: mime.getType(args.FilePath) || undefined, // set the MIME type of the file
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
