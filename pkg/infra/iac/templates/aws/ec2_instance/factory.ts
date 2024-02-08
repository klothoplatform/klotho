import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    InstanceProfile: aws.iam.InstanceProfile
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnet: aws.ec2.Subnet
    AMI: aws.ec2.Ami
    InstanceType: string
}

function create(args: Args): aws.ec2.Instance {
    return new aws.ec2.Instance(args.Name, {
        ami: args.AMI.id,
        iamInstanceProfile: args.InstanceProfile,
        vpcSecurityGroupIds: args.SecurityGroups.map((sg) => sg.id),
        subnetId: args.Subnet.id,
        instanceType: args.InstanceType,
    })
}

function properties(object: aws.ec2.Instance, args: Args) {
    return {
        Id: object.id,
    }
}
