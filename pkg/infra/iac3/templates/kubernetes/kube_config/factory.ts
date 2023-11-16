import * as pulumi from '@pulumi/pulumi'
import { TemplateWrapper } from '../../wrappers'

interface Args {
    ApiVersion: string
    Kind: string
    Name: string
    Clusters: TemplateWrapper<any[]>
    Contexts: any[]
    Users: any[]
    CurrentContext: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi.Output<string> {
    return pulumi.jsonStringify({
        apiVersion: args.ApiVersion,
        clusters: args.Clusters,
        contexts: args.Contexts,
        'current-context': args.CurrentContext,
        kind: args.Kind,
        users: args.Users,
    })
}