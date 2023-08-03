import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    FilePath: string
    Transformations?: Record<string, pulumi.Output<string>>
    clusterProvider: pulumi_k8s.Provider
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.yaml.ConfigFile {
    return new pulumi_k8s.yaml.ConfigFile(
        args.Name,
        {
            file: args.FilePath,
            //TMPL {{- if .Transformations.Raw }}
            transformations: [
                (obj: any, opts: pulumi.CustomResourceOptions) => {
                    //TMPL {{- range $key, $value := .Transformations.Raw }}
                    //TMPL obj.{{ $key }} = {{ handleIaCValue $value }}
                    //TMPL {{- end }}
                },
            ],
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
            provider: args.clusterProvider,
        }
    )
}
