import * as pulumi_k8s from '@pulumi/kubernetes'

export const installMetricsServer = (
    clusterName: string,
    provider: pulumi_k8s.Provider,
    dependsOn?
): pulumi_k8s.helm.v3.Release => {
    // Declare the ALBIngressController in 1 step with the Helm Chart.
    return new pulumi_k8s.helm.v3.Release(
        `${clusterName}-metrics-server`,
        {
            name: 'metrics-server',
            chart: 'metrics-server',
            repositoryOpts: { repo: 'https://kubernetes-sigs.github.io/metrics-server/' },
        },
        { provider, dependsOn, deleteBeforeReplace: true }
    )
}
