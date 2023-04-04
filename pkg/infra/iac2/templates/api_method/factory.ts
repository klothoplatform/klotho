import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Resource: aws.apigateway.Resource
    HttpMethod: string
    RequestParameters: Record<string, boolean>
    Authorization: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Method {
    return new aws.apigateway.Method(args.Name, {
        restApi: args.RestApi.id,
        resourceId: args.Resource.id,
        httpMethod: args.HttpMethod,
        authorization: args.Authorization,
        requestParameters: args.RequestParameters,
    })
}
