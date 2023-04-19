import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Subnet: aws.ec2.Subnet
    RouteTable: aws.ec2.RouteTable
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.RouteTableAssociation {
    return new aws.ec2.RouteTableAssociation(args.Name, {
        subnetId: args.Subnet.id,
        routeTableId: args.RouteTable.id,
    })
}
