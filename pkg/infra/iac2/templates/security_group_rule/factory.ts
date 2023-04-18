import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    FromPort: number
    ToPort: number
    Protocol: string
    CidrBlocks: string[]
    Cluster: aws.eks.Cluster
    Type: string
}

function create(args: Args): aws.ec2.SecurityGroupRule {
    return new aws.ec2.SecurityGroupRule(args.Name, {
        type: args.Type,
        cidrBlocks: args.CidrBlocks,
        fromPort: args.FromPort,
        protocol: args.Protocol,
        toPort: args.ToPort,
        securityGroupId: args.Cluster.vpcConfig.clusterSecurityGroupId,
    })
}
