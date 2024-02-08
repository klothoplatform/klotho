import * as aws from '@pulumi/aws'

interface Args {
    Name: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Eip {
    return new aws.ec2.Eip(args.Name, {})
}
