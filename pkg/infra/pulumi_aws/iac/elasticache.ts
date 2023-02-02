import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { Resource } from '../deploylib'
import * as validators from './sanitization/aws/elasticache'
import { sanitized } from './sanitization/sanitizer'

const ELASTICACHE_ENGINE = 'redis'

const sanitizeClusterName = (appName: string, dbName: string): string => {
    let cluster = `${appName}-${dbName}`
    if (cluster.length >= 40) {
        const oldCluster = cluster
        cluster = ''
        if (appName.length > 20) {
            cluster += appName.substring(0, 10) + appName.substring(appName.length - 10)
        } else {
            cluster += appName
        }
        cluster += '-'
        if (dbName.length > 19) {
            cluster += dbName.substring(0, 10) + dbName.substring(dbName.length - 9)
        } else {
            cluster += dbName
        }
        cluster = cluster.toLowerCase()

        console.info(
            `Abbreviating cluster name '${oldCluster}' to '${cluster}' due to exceeding length limits (${oldCluster.length} >= 20)`
        )
    }
    return cluster.toLocaleLowerCase().replace('_', '-')
}

export const setupElasticacheCluster = (
    dbName: string,
    args: Partial<aws.elasticache.ClusterArgs>,
    topology: any,
    protect: boolean,
    connectionString: Map<string, pulumi.Output<string>>,
    subnetGroupName: pulumi.Output<string>,
    securityGroupIds: pulumi.Output<string>[],
    appName: string
) => {
    const logGroupName = `/aws/elasticache/${appName}-${dbName}-persist-redis`
    let cloudwatchGroup = new aws.cloudwatch.LogGroup(`persist-redis-${dbName}-lg`, {
        name: `${logGroupName}`,
        retentionInDays: 0,
    })

    // TODO: look into removing sanitizeClusterName when making other breaking changes to resource names
    const clusterName = sanitized(
        validators.cacheCluster.cacheClusterIdValidation()
    )`${sanitizeClusterName(appName, dbName)}`
    // create the db resources
    const redis = new aws.elasticache.Cluster(
        clusterName,
        {
            engine: ELASTICACHE_ENGINE,
            clusterId: clusterName,
            ...args,
            logDeliveryConfigurations: [
                {
                    destination: cloudwatchGroup.name,
                    destinationType: 'cloudwatch-logs',
                    logFormat: 'text',
                    logType: 'slow-log',
                },
                {
                    destination: cloudwatchGroup.name,
                    destinationType: 'cloudwatch-logs',
                    logFormat: 'json',
                    logType: 'engine-log',
                },
            ],
            subnetGroupName,
            securityGroupIds,
        },
        { protect }
    )

    topology.topologyIconData.forEach((resource) => {
        if (resource.kind == Resource.redis_node) {
            topology.topologyEdgeData.forEach((edge) => {
                var id = resource.id
                if (edge.target == id && id == `${dbName}_${Resource.redis_node}`) {
                    connectionString.set(
                        `${id}_host`,
                        pulumi.interpolate`${redis.cacheNodes.apply((nodes) => nodes[0].address)}`
                    )
                    connectionString.set(
                        `${id}_port`,
                        pulumi.interpolate`${redis.cacheNodes.apply((nodes) => nodes[0].port)}`
                    )

                    // Set another copy of the connection string for helm env vars in case they are used
                    const envVar = `${resource.title}${resource.kind}`
                        .toUpperCase()
                        .replace(/[^a-z0-9]/gi, '')
                    connectionString.set(
                        `${envVar}HOST`,
                        pulumi.interpolate`${redis.cacheNodes.apply((nodes) => nodes[0].address)}`
                    )
                    connectionString.set(
                        `${envVar}PORT`,
                        pulumi.interpolate`${redis.cacheNodes.apply((nodes) => nodes[0].port)}`
                    )
                }
            })
        }
    })
}
