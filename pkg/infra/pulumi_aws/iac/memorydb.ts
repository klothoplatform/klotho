import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { Resource } from '../deploylib'
import { sanitized } from './sanitization/sanitizer'
import { cacheCluster } from './sanitization/aws/memorydb'

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

export const setupMemoryDbCluster = (
    dbName: string,
    args: Partial<aws.memorydb.ClusterArgs>,
    topology: any,
    protect: boolean,
    connectionString: Map<string, pulumi.Output<string>>,
    subnetGroupName: pulumi.Output<string>,
    securityGroupIds: pulumi.Output<string>[],
    appName: string
) => {
    // TODO: look into removing sanitizeClusterName when making other breaking changes to resource names
    const clusterName = sanitized(cacheCluster.clusterNameValidation())`${sanitizeClusterName(
        appName,
        dbName
    )}`
    const memdbCluster = new aws.memorydb.Cluster(
        clusterName,
        {
            ...args,
            name: clusterName,
            aclName: 'open-access',
            nodeType: 'db.t4g.small',
            securityGroupIds,
            snapshotRetentionLimit: 7,
            subnetGroupName,
        },
        { protect }
    )

    topology.topologyIconData.forEach((resource) => {
        if (resource.kind == Resource.redis_cluster) {
            topology.topologyEdgeData.forEach((edge) => {
                var id = resource.id
                if (edge.target == id && id == `${dbName}_${Resource.redis_cluster}`) {
                    connectionString.set(
                        `${id}_host`,
                        pulumi.interpolate`${memdbCluster.clusterEndpoints.apply(
                            (endpoint) => endpoint[0].address
                        )}`
                    )
                    connectionString.set(
                        `${id}_port`,
                        pulumi.interpolate`${memdbCluster.clusterEndpoints.apply(
                            (endpoint) => endpoint[0].port
                        )}`
                    )
                }
            })
        }
    })
}
