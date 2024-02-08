import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    FilePath: string
    Transformations?: Record<string, pulumi.Output<string>>
    Provider: pulumi_k8s.Provider
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.yaml.ConfigFile {
    return new pulumi_k8s.yaml.ConfigFile(
        args.Name,
        {
            file: args.FilePath,
            //TMPL {{- if .Transformations }}
            transformations: args.Transformations,
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
            provider: args.Provider,
        }
    )
}
