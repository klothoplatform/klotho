import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Resource: aws.apigateway.Resource
    Method: aws.apigateway.Method
    IntegrationHttpMethod: string
    Type: string
    ConnectionType: string
    VpcLink: aws.apigateway.VpcLink
    RequestParameters: Record<string, string>
    Uri: pulumi.Output<string>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Integration {
    return new aws.apigateway.Integration(args.Name, {
        restApi: args.RestApi.id,
        resourceId: args.Resource.id,
        httpMethod: args.Method.httpMethod,
        integrationHttpMethod: args.Method.httpMethod,
        type: args.Type,
        //TMPL {{ if .ConnectionType.Raw }}
        connectionType: args.ConnectionType,
        //TMPL {{ end }}
        //TMPL {{ if .VpcLink.Raw }}
        connectionId: args.VpcLink.id,
        //TMPL {{ end }}
        uri: args.Uri,
        requestParameters: args.RequestParameters,
    })
}
