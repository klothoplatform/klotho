import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Deployment: aws.apigateway.Deployment
    StageName: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Stage {
    return new aws.apigateway.Stage(args.Name, {
        deployment: args.Deployment.id,
        restApi: args.RestApi.id,
        stageName: args.StageName,
    })
}
