import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    EmailIdentity: string
}

function create(args: Args): aws.ses.EmailIdentity {
    return new aws.ses.EmailIdentity(args.Name, {
        email: args.EmailIdentity,
    })
}

function properties(object: aws.ses.EmailIdentity, args: Args) {
    return {
        Arn: object.arn,
    }
}
