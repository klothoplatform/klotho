import * as k8s from '@pulumi/kubernetes'
import * as pulumi from '@pulumi/pulumi'

export interface HorizontalPodAutoscalingArgs {
    deploymentName: string
    minReplicas: number
    maxReplicas: number
    metrics: k8s.types.input.autoscaling.v2.MetricSpec[]
    provider: k8s.Provider
    dependsOn: pulumi.Resource[]
}

export const autoScaleDeployment = (args: HorizontalPodAutoscalingArgs) => {
    const { deploymentName, minReplicas, maxReplicas, metrics, provider, dependsOn } = args
    return new k8s.autoscaling.v2.HorizontalPodAutoscaler(
        `${deploymentName}-Autoscaler`,
        {
            metadata: {
                name: deploymentName,
            },
            spec: {
                scaleTargetRef: {
                    apiVersion: 'apps/v1',
                    kind: 'Deployment',
                    name: deploymentName,
                },
                minReplicas,
                maxReplicas,
                metrics,
            },
        },
        { provider, dependsOn }
    )
}

export const createPodAutoScalerResourceMetric = (
    metricName: string,
    type: 'Utilization' | 'AverageValue',
    value
): k8s.types.input.autoscaling.v2.MetricSpec => {
    const metric: k8s.types.input.autoscaling.v2.MetricSpec = {
        type: 'Resource',
        resource: {
            name: metricName,
            target: {
                type,
            },
        },
    }
    switch (type) {
        case 'Utilization':
            metric.resource!['target']['averageUtilization'] = value
            break
        case 'AverageValue':
            metric.resource!['target']['averageValue'] = value
            break
    }
    return metric
}
