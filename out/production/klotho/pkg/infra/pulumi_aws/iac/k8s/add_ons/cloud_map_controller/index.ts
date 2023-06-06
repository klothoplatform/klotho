import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import * as pulumi_k8s from '@pulumi/kubernetes'

export const installCloudMapController = (
    clusterName: string,
    provider: pulumi_k8s.Provider,
    dependsOn?
): pulumi_k8s.kustomize.Directory => {
    const cloudMapPlugin = new pulumi_k8s.kustomize.Directory(
        `${clusterName}-cloud-map-plugin`,
        {
            directory:
                'https://github.com/aws/aws-cloud-map-mcs-controller-for-k8s/config/controller_install_release',
        },
        { provider, dependsOn }
    )

    return cloudMapPlugin
}

export const createClusterSet = (
    clusterName: string,
    provider: pulumi_k8s.Provider,
    dependsOn?
): pulumi_k8s.yaml.ConfigFile => {
    return new pulumi_k8s.yaml.ConfigFile(
        `${clusterName}-cluster-set`,
        {
            file: './iac/k8s/add_ons/cloud_map_controller/cloudmap_cluster_set.yaml',
            transformations: [
                (obj: any, opts: pulumi.CustomResourceOptions) => {
                    if (obj.metadata.name == 'cluster.clusterset.k8s.io') {
                        obj.spec.value = clusterName
                    }
                    if (obj.metadata.name == 'clusterset.k8s.io') {
                        obj.spec.value = `${clusterName}-set`
                    }
                },
            ],
        },
        { provider, dependsOn }
    )
}

export const createServiceExport = (
    execUnit,
    namespaceId,
    provider: pulumi_k8s.Provider,
    parent,
    dependsOn
) => {
    return new pulumi_k8s.yaml.ConfigFile(
        `${execUnit}-service-export`,
        {
            file: './iac/k8s/add_ons/cloud_map_controller/cloudmap_export_service.yaml',
            transformations: [
                // Make every service private to the cluster, i.e., turn all services into ClusterIP instead of LoadBalancer.
                (obj: any, opts: pulumi.CustomResourceOptions) => {
                    obj.metadata = { name: execUnit, namespace: namespaceId }
                },
            ],
        },
        { provider, parent, dependsOn }
    )
}
