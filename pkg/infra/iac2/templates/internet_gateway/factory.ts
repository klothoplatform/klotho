import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.InternetGateway {
    return new aws.ec2.InternetGateway(args.Name, {
        vpcId: args.Vpc.id,
    })
}
