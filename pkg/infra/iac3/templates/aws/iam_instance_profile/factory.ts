import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Role: aws.iam.Role
}

function create(args: Args): aws.iam.InstanceProfile {
    return new aws.iam.InstanceProfile(args.Name, {
        role: args.Role,
    })
}
