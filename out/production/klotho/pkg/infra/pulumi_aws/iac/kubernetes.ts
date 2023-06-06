import * as k8s from '@pulumi/kubernetes'
import * as pulumi from '@pulumi/pulumi'

export interface DeploymentArgs {
    name: string
    image: string
    replicas?: number
    appLabels?: { [key: string]: string }
    resources?: k8s.types.input.core.v1.ResourceRequirements
    env?: pulumi.Input<pulumi.Input<k8s.types.input.core.v1.EnvVar>[]>
    serviceAccountName?: pulumi.Output<string>
    nodeSelector?: { [key: string]: string | pulumi.Output<string> }
    k8sProvider: k8s.Provider
    parent
}

export const createDeployment = (args: DeploymentArgs): k8s.apps.v1.Deployment => {
    const {
        name,
        image,
        replicas,
        appLabels,
        resources,
        env,
        serviceAccountName,
        k8sProvider,
        parent,
        nodeSelector,
    } = args

    return new k8s.apps.v1.Deployment(
        name.replace('-', '').toLowerCase(),
        {
            metadata: {
                name,
                labels: appLabels,
            },
            spec: {
                strategy: {
                    type: 'RollingUpdate',
                    rollingUpdate: {
                        maxSurge: 1,
                        maxUnavailable: 1,
                    },
                },
                selector: { matchLabels: appLabels },
                replicas,
                template: {
                    metadata: { labels: appLabels },
                    spec: {
                        nodeSelector,
                        serviceAccountName,
                        containers: [
                            {
                                name: name.replace('-', '').toLowerCase(),
                                image,
                                env,
                                resources,
                            },
                        ],
                    },
                },
            },
        },
        { provider: k8sProvider, parent }
    )
}

export const createService = (
    execUnit: string,
    k8sProvider: k8s.Provider,
    appLabels: { [key: string]: string },
    annotations,
    stickinessTimeout: number,
    parent,
    dependsOn
): k8s.core.v1.Service => {
    let sessionAffinityFields = {}
    if (stickinessTimeout > 0) {
        sessionAffinityFields = {
            sessionAffinity: 'ClientIP',
            sessionAffinityConfig: {
                clientIP: {
                    timeoutSeconds: stickinessTimeout,
                },
            },
        }
    }

    return new k8s.core.v1.Service(
        execUnit.replace('-', '').toLowerCase(),
        {
            metadata: {
                name: execUnit,
                annotations: annotations,
                labels: appLabels,
            },
            spec: {
                ports: [
                    {
                        port: 80,
                        protocol: 'TCP',
                        targetPort: 3000,
                    },
                ],
                selector: appLabels,
                ...sessionAffinityFields,
            },
        },
        { provider: k8sProvider, parent, dependsOn }
    )
}
