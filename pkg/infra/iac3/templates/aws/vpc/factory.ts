import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    CidrBlock: string
    EnableDnsHostnames: boolean
    EnableDnsSupport: boolean
    Id?: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Vpc {
    return new aws.ec2.Vpc(args.Name, {
        cidrBlock: args.CidrBlock,
        enableDnsHostnames: args.EnableDnsHostnames,
        enableDnsSupport: args.EnableDnsSupport,
        tags: {
            Name: args.Name,
        },
    })
}

function properties(object: aws.ec2.Vpc, args: Args) {
    return {
        Id: object.id,
    }
}

function importResource(args: Args): aws.ec2.Vpc {
    return aws.ec2.Vpc.get(args.Name, args.Id)
}
