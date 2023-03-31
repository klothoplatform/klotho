import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
}

function create(args: Args): aws.ec2.SecurityGroup {
    return new aws.ec2.SecurityGroup(args.Name, {
        name: args.Name,
        vpcId: args.Vpc.id,
        egress: [
            {
                cidrBlocks: ['0.0.0.0/0'],
                description: 'Allows all outbound IPv4 traffic.',
                fromPort: 0,
                protocol: '-1',
                toPort: 0,
            },
        ],
        ingress: [
            {
                description:
                    'Allows inbound traffic from network interfaces and instances that are assigned to the same security group.',
                fromPort: 0,
                protocol: '-1',
                self: true,
                toPort: 0,
            },
            {
                description: 'For EKS control plane',
                cidrBlocks: ['0.0.0.0/0'],
                fromPort: 9443,
                protocol: 'TCP',
                toPort: 9443,
            },
            {
                description: 'For private subnets internally',
                cidrBlocks: [args.Vpc.cidrBlock],
                fromPort: 0,
                protocol: '-1',
                toPort: 0,
            },
        ],
    })
}
