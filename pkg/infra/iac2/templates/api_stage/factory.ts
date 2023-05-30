import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Deployment: aws.apigateway.Deployment
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Stage {
    return new aws.apigateway.Stage(args.Name, {
        deployment: args.Deployment.id,
        restApi: args.RestApi.id,
        stageName: args.Deployment.stageName.apply(v => v ?? "UNSPECIFIED_STAGE"),
    })
}
