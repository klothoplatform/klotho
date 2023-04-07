import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    KubeConfig: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.Provider {
    return new pulumi_k8s.Provider(args.Name, {
        kubeconfig: args.KubeConfig,
    })
}

// TODO replace this factory with a more fleshed out implementation
