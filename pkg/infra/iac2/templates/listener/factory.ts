import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Port: number
    Protocol: string
    LoadBalancer: aws.lb.LoadBalancer
    DefaultActions: aws.types.input.lb.ListenerDefaultAction[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.Listener {
    return new aws.lb.Listener(args.Name, {
        loadBalancerArn: args.LoadBalancer.arn,
        defaultActions: args.DefaultActions,
        port: args.Port,
        protocol: args.Protocol,
    })
}
