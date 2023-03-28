import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    CidrBlock: string
    Vpc: aws.ec2.Vpc
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Subnet {
    return new aws.ec2.Subnet(args.Name, {
        vpcId: args.Vpc.main.id,
        cidrBlock: args.CidrBlock,
    })
}
