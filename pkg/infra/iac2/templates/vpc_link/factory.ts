import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Target: aws.lb.LoadBalancer
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.VpcLink {
    return new aws.apigateway.VpcLink(args.Name, {
        targetArn: args.Target.arn,
    })
}
