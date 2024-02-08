import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Bucket: aws.s3.Bucket
    Policy: ModelCaseWrapper<aws.iam.PolicyDocument>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.s3.BucketPolicy {
    return new aws.s3.BucketPolicy(args.Name, {
        bucket: args.Bucket.id,
        policy: args.Policy,
    })
}
