import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    ElasticIp: aws.ec2.Eip
    Subnet: aws.ec2.Subnet
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.NatGateway {
    return new aws.ec2.NatGateway(args.Name, {
        allocationId: args.ElasticIp.id,
        subnetId: args.Subnet.id,
    })
}
