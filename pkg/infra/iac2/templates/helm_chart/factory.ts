import * as pulumi_k8s from '@pulumi/kubernetes'

interface Args {
    Name: string
    Directory: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): pulumi_k8s.helm.v3.Chart {
    return new pulumi_k8s.helm.v3.Chart(args.Name, {
        path: `./charts/${args.Directory}`,
    })
}
