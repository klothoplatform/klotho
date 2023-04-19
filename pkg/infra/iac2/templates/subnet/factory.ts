import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    CidrBlock: string
    Vpc: aws.ec2.Vpc
    AvailabilityZone: pulumi.Output<string>
    MapPublicIpOnLaunch: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Subnet {
    return new aws.ec2.Subnet(args.Name, {
        vpcId: args.Vpc.id,
        cidrBlock: args.CidrBlock,
        availabilityZone: args.AvailabilityZone,
        mapPublicIpOnLaunch: args.MapPublicIpOnLaunch,
        tags: {
            Name: args.Name,
        },
    })
}
