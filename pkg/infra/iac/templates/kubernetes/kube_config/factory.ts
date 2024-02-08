import * as pulumi from '@pulumi/pulumi'
import { TemplateWrapper } from '../../wrappers'

interface Args {
    apiVersion: string
    kind: string
    name: string
    clusters: TemplateWrapper<any[]>
    contexts: any[]
    users: any[]
    currentContext: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi.Output<string> {
    return pulumi.jsonStringify({
        apiVersion: args.apiVersion,
        clusters: args.clusters,
        contexts: args.contexts,
        'current-context': args.currentContext,
        kind: args.kind,
        users: args.users,
    })
}
