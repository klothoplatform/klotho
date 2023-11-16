import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Resource: aws.apigateway.Resource
    Method: aws.apigateway.Method
    IntegrationHttpMethod: string
    Type: string
    ConnectionType: string
    VpcLink: aws.apigateway.VpcLink
    RequestParameters: ModelCaseWrapper<Record<string, string>>
    Uri: pulumi.Output<string>
    Route: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Integration {
    return new aws.apigateway.Integration(
        args.Name,
        {
            restApi: args.RestApi.id,
            //TMPL {{- if .Resource }}
            resourceId: args.Resource.id,
            //TMPL {{- else }}
            //TMPL resourceId: args.RestApi.rootResourceId,
            //TMPL {{- end }}
            httpMethod: args.Method.httpMethod,
            integrationHttpMethod: args.IntegrationHttpMethod,
            type: args.Type,
            //TMPL {{- if .ConnectionType }}
            connectionType: args.ConnectionType,
            //TMPL {{- end }}
            //TMPL {{- if .VpcLink }}
            connectionId: args.VpcLink.id,
            //TMPL {{- end }}
            uri: args.Uri,
            //TMPL {{- if .RequestParameters }}
            requestParameters: args.RequestParameters,
            //TMPL {{- end }}
        },
        { parent: args.Method }
    )
}

function properties(object: aws.apigateway.Integration, args: Args) {
    return {
        LbUri: pulumi.interpolate`http://${
            (args.Target as aws.lb.LoadBalancer).dnsName
        }${args.Route.replace('+', '')}`,
    }
}
