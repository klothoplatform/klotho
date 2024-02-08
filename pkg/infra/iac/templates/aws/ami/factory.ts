import * as aws from '@pulumi/aws'

interface Args {
    Name: string
}

function create(args: Args): aws.ec2.Ami {
    return new aws.ec2.Ami(args.Name, {})
}
