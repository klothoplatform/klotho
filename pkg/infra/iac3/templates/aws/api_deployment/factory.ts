import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    RestApi: aws.apigateway.RestApi
    Triggers: Record<string, string>
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.apigateway.Deployment {
    return new aws.apigateway.Deployment(
        args.Name,
        {
            restApi: args.RestApi.id,
            //TMPL {{- if .Triggers }}
            triggers: args.Triggers,
            //TMPL {{- end}}
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}
