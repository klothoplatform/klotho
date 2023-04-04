import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    PathPart: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Resource {
    return new aws.apigateway.Resource(args.Name, {
        restApi: args.RestApi.id,
        parentId: args.RestApi.rootResourceId,
        pathPart: args.PathPart,
    })
}
