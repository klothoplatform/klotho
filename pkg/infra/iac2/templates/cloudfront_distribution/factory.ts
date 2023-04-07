import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Origins: pulumi.Input<aws.types.input.cloudfront.DistributionOrigin>[]
    CloudfrontDefaultCertificate: boolean
    Enabled: boolean
    DefaultCacheBehavior: aws.types.input.cloudfront.DistributionDefaultCacheBehavior
    Restrictions: aws.types.input.cloudfront.DistributionRestrictions
    DefaultRootObject: string
}

function create(args: Args): aws.cloudfront.Distribution {
    return new aws.cloudfront.Distribution(args.Name, {
        origins: args.Origins,
        enabled: args.Enabled,
        viewerCertificate: {
            cloudfrontDefaultCertificate: args.CloudfrontDefaultCertificate,
        },
        defaultCacheBehavior: args.DefaultCacheBehavior,
        restrictions: args.Restrictions,
        defaultRootObject: args.DefaultRootObject,
    })
}
