import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    SanitizedName: string
    Directory?: string
    Chart?: string
    Repo?: string
    Values?: Record<string, pulumi.Output<any>>
    Version?: string
    Namespace?: string
    clusterProvider: pulumi_k8s.Provider
    dependsOn: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.helm.v3.Chart {
    return new pulumi_k8s.helm.v3.Chart(
        args.SanitizedName,
        {
            //TMPL {{- if .Chart }}
            chart: args.Chart,
            //TMPL {{- end }}
            //TMPL {{- if .Repo }}
            fetchOpts: {
                repo: args.Repo,
            },
            //TMPL {{- end }}
            //TMPL {{- if and (not .Chart) .Directory }}
            path: args.Directory,
            //TMPL {{- end }}
            //TMPL {{- if .Namespace }}
            namespace: args.Namespace,
            //TMPL {{- end }}
            //TMPL {{- if .Version }}
            version: args.Version,
            //TMPL {{- end }}
            //TMPL {{- if .Values }}
            values: args.Values,
            //TMPL {{- end }}
        },
        {
            provider: args.clusterProvider,
            dependsOn: args.dependsOn,
        }
    )
}
