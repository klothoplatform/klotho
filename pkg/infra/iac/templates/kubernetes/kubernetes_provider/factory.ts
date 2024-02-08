import * as pulumi_k8s from '@pulumi/kubernetes'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    KubeConfig: pulumi.Output<string>
    EnableServerSideApply: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.Provider {
    return new pulumi_k8s.Provider(args.Name, {
        kubeconfig: args.KubeConfig,
        //TMPL {{- if .EnableServerSideApply }}
        enableServerSideApply: args.EnableServerSideApply,
        //TMPL {{- end }}
    })
}
