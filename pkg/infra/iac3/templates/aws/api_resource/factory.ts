import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    PathPart: string
    ParentResource: aws.apigateway.Resource
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Resource {
    return new aws.apigateway.Resource(
        args.Name,
        {
            restApi: args.RestApi.id,
            //TMPL {{- if .ParentResource }}
            parentId: args.ParentResource.id,
            //TMPL {{- else}}
            //TMPL parentId: args.RestApi.rootResourceId,
            //TMPL {{- end }}
            pathPart: args.PathPart,
        },
        { parent: args.RestApi }
    )
}
