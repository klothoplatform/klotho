import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Policy: aws.iam.Policy
    Role: aws.iam.Role
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.RolePolicyAttachment {
    return new aws.iam.RolePolicyAttachment(args.Name, {
        policyArn: args.Policy.arn,
        role: args.Role,
    })
}
