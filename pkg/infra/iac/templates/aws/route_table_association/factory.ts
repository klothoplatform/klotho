import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Id?: string
    SubnetId: string
    RouteTableId: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.RouteTableAssociation {
    return new aws.ec2.RouteTableAssociation(args.Name, {
        subnetId: args.SubnetId,
        routeTableId: args.RouteTableId,
    })
}

function properties(object: aws.ec2.RouteTableAssociation, args: Args) {
    return {
        Id: object.id,
    }
}

function importResource(args: Args): aws.ec2.RouteTableAssociation {
    return aws.ec2.RouteTableAssociation.get(args.Name, `${args.SubnetId}/${args.RouteTableId}`)
}
