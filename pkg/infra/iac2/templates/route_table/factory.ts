import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    Routes: aws.types.input.ec2.RouteTableRoute[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.RouteTable {
    return new aws.ec2.RouteTable(args.Name, {
        vpcId: args.Vpc.id,
        routes: args.Routes,
    })
}
