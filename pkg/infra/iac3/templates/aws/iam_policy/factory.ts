import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Policy: aws.iam.PolicyDocument
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Policy {
    return new aws.iam.Policy(args.Name, {
        policy: args.Policy,
    })
}
