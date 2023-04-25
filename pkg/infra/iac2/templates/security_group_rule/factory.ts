import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Description: string
    FromPort: number
    ToPort: number
    Protocol: string
    CidrBlocks: string[]
    SecurityGroupId: pulumi.Input<string>
    Type: string
}

function create(args: Args): aws.ec2.SecurityGroupRule {
    return new aws.ec2.SecurityGroupRule(args.Name, {
        description: args.Description,
        type: args.Type,
        cidrBlocks: args.CidrBlocks,
        fromPort: args.FromPort,
        protocol: args.Protocol,
        toPort: args.ToPort,
        securityGroupId: args.SecurityGroupId,
    })
}
