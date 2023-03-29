import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    CidrBlock: string
    EnableDnsHostnames: boolean
    EnableDnsSupport: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Vpc {
    return new aws.ec2.Vpc(args.Name, {
        cidrBlock: args.CidrBlock,
        enableDnsHostnames: args.EnableDnsHostnames,
        enableDnsSupport: args.EnableDnsSupport,
    })
}
