import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    Tags: Record<string, string>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.rds.SubnetGroup {
    return new aws.rds.SubnetGroup(args.Name, {
        subnetIds: args.Subnets.map((subnet) => subnet.id),
        tags: args.Tags,
    })
}
