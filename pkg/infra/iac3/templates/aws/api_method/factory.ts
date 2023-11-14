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
    return new aws.apigateway.Method(
        args.Name,
        {
            restApi: args.RestApi.id,
            //TMPL {{- if .Resource }}
            resourceId: args.Resource.id,
            //TMPL {{- else}}
            //TMPL resourceId: args.RestApi.rootResourceId,
            //TMPL {{- end }}
            httpMethod: args.HttpMethod,
            authorization: args.Authorization,
            //TMPL {{- if .RequestParameters }}
            requestParameters: args.RequestParameters,
            //TMPL {{- end }}
        },
        {
            //TMPL {{- if .Resource }}
            parent: args.Resource,
            //TMPL {{- else }}
            //TMPL parent: args.RestApi
            //TMPL {{- end }}
        }
    )
}
