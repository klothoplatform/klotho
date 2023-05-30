import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Port: number
    Protocol: string
    Vpc: aws.ec2.Vpc
    TargetType: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.TargetGroup {
    return new aws.lb.TargetGroup(args.Name, {
        port: args.Port,
        protocol: args.Protocol,
        targetType: args.TargetType,
        vpcId: args.Vpc.id,
    })
}
