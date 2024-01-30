import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.servicediscovery.PrivateDnsNamespace {
    return new aws.servicediscovery.PrivateDnsNamespace(args.Name, {
        vpc: args.Vpc.id,
    })
}

function properties(object: aws.servicediscovery.PrivateDnsNamespace, args: Args) {
    return {
        Id: object.id,
    }
}
