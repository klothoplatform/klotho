import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    BinaryMediaTypes: string[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.RestApi {
    return new aws.apigateway.RestApi(args.Name, {
        binaryMediaTypes: args.BinaryMediaTypes,
    })
}
