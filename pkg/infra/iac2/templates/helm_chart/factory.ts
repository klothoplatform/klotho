import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    Directory?: pulumi.Input<string>
    Chart?: pulumi.Input<string>
    FetchOpts?: pulumi.Input<pulumi_k8s.helm.v3.FetchOpts>
    Values?: pulumi.Inputs
    Version?: pulumi.Input<string>
    Namespace?: pulumi.Input<string>
    KubernetesProvider: pulumi_k8s.Provider
    dependsOn: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.helm.v3.Chart {
    return new pulumi_k8s.helm.v3.Chart(
        args.Name,
        {
            //TMPL {{- if .Chart.Raw }}
            chart: args.Chart,
            //TMPL {{- end }}
            //TMPL {{- if and (.Chart.Raw) (.FetchOpts.Raw) }}
            fetchOpts: args.FetchOpts,
            //TMPL {{- end }}
            //TMPL {{- if not .Chart.Raw }}
            path: `./charts/${args.Directory}`,
            //TMPL {{- end }}
            //TMPL {{- if .Namespace.Raw }}
            namespace: args.Namespace,
            //TMPL {{- end }}
            //TMPL {{- if .Version.Raw }}
            version: args.Version,
            //TMPL {{- end }}
            //TMPL {{- if .Values.Raw }}
            values: args.Values,
            //TMPL {{- end }}
        },
        {
            provider: args.KubernetesProvider,
            //TMPL {{- if .dependsOn.Raw }}
            dependsOn: args.dependsOn,
            //TMPL {{- end }}
        }
    )
}
