import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Listener: aws.lb.Listener
    Certificate: aws.acm.Certificate
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.LoadBalancer {
    return new aws.lb.ListenerCertificate('exampleListenerCertificate', {
        listenerArn: args.Listener.arn,
        certificateArn: args.Certificate.arn,
    })
}
