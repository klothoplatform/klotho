import * as pulumi_k8s from '@pulumi/kubernetes'
import { Eks } from '../../../eks'

export const installFluentBitForCW = (eks: Eks, provider) => {
    const namespaceName = 'amazon-cloudwatch'
    const label = { name: namespaceName }
    const cwNamespace = eks.createNamespace(namespaceName, label)

    const configMap = new pulumi_k8s.core.v1.ConfigMap(
        `${eks.clusterName}-fluent-bit-cluster-info`,
        {
            metadata: {
                name: 'fluent-bit-cluster-info',
                namespace: namespaceName,
            },
            data: {
                'cluster.name': eks.clusterName,
                'logs.region': eks.region,
                'http.server': 'On',
                'http.port': '2020',
                'read.head': 'Off',
                'read.tail': 'On',
            },
        },
        { provider, dependsOn: [cwNamespace] }
    )
    new pulumi_k8s.yaml.ConfigFile(
        `${eks.clusterName}-FluentBitDriver`,
        {
            file: 'https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/fluent-bit/fluent-bit.yaml',
        },
        {
            provider,
            dependsOn: [cwNamespace, configMap],
        }
    )
}

export const enableFargateLogging = (eks: Eks, provider) => {
    const ns = 'aws-observability'
    const labels = { 'aws-observability': 'enabled' }
    eks.createNamespace(ns, labels)

    const configMap = new pulumi_k8s.core.v1.ConfigMap(
        'aws-observability-configmap',
        {
            metadata: {
                name: 'aws-logging',
                namespace: ns,
            },
            data: {
                'output.conf': `[OUTPUT]
                Name cloudwatch_logs
                Match   *
                region ${eks.region}
                log_group_name fluent-bit-cloudwatch
                log_stream_prefix from-fluent-bit-
                auto_create_group true
                log_key log`,

                'parsers.conf': `[PARSER]
                Name crio
                Format Regex
                Regex ^(?<time>[^ ]+) (?<stream>stdout|stderr) (?<logtag>P|F) (?<log>.*)$
                Time_Key    time
                Time_Format %Y-%m-%dT%H:%M:%S.%L%z`,

                'filters.conf': `[FILTER]
                Name parser
                Match *
                Key_name log
                Parser crio`,
            },
        },
        { provider }
    )
}
