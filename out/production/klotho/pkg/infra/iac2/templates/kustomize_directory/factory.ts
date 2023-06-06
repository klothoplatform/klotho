import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    Directory: string
    ClustersProvider: pulumi_k8s.Provider
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.kustomize.Directory {
    return new pulumi_k8s.kustomize.Directory(
        args.Name,
        {
            directory: args.Directory,
        },
        {
            dependsOn: args.dependsOn,
            provider: args.ClustersProvider,
        }
    )
}
