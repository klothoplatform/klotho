import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Directory?: string
    Chart?: string
    Repo?: string
    Values?: ModelCaseWrapper<Record<string, pulumi.Output<any>>>
    Version?: string
    Namespace?: string
    Provider: pulumi_k8s.Provider
    dependsOn: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.helm.v3.Chart {
    return new pulumi_k8s.helm.v3.Chart(
        args.Name,
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
            provider: args.Provider,
            dependsOn: args.dependsOn,
        }
    )
}
