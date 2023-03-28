import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    Region: string
    ServiceName: string
    VpcEndpointType: string
    Subnets: aws.ec2.Subnet[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.VpcEndpoint {
    return new aws.ec2.VpcEndpoint(args.Name, {
        vpcId: args.Vpc.id,
        serviceName: `com.amazonaws.${args.Region}.${args.ServiceName}`,
        vpcEndpointType: args.VpcEndpointType,
        privateDnsEnabled: true,
        subnetIds: args.Subnets.map((x) => x.id),
        routeTableIds: args.Vpc.defaultRouteTableId,
    })
}
