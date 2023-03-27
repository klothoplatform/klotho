import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    AssumeRolePolicyDoc: string
    ManagedPolicyArns: string[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Role {
    return new aws.iam.Role(args.Name, {
        assumeRolePolicy: JSON.parse(args.AssumeRolePolicyDoc),
    })
}
