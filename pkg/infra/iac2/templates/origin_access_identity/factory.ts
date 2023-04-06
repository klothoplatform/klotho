import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Comment: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.cloudfront.OriginAccessIdentity {
    return new aws.cloudfront.OriginAccessIdentity(args.Name, {
        comment: args.Comment,
    })
}
