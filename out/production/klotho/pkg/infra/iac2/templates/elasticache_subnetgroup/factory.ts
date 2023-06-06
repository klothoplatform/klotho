import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
}

function create(args: Args): aws.elasticache.SubnetGroup {
    return new aws.elasticache.SubnetGroup(args.Name, {
        subnetIds: args.Subnets.map((sg) => sg.id),
    })
}
