import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { CloudCCLib, Resource, kloConfig } from '../deploylib'
import * as validators from './sanitization/aws/elasticache'
import { sanitized, hash as h } from './sanitization/sanitizer'
import AwsSanitizer from './sanitization/aws'

const ELASTICACHE_ENGINE = 'redis'

export class Elasticache {
    constructor() {}

    static setupElasticacheCluster(
        lib: CloudCCLib,
        dbName: string,
        args: Partial<aws.elasticache.ClusterArgs>
    ) {
        const elasticacheImports: { [key: string]: string } | undefined =
            kloConfig.getObject('elasticache')
        // TODO: look into removing sanitizeClusterName when making other breaking changes to resource names
        const clusterName = sanitized(
            validators.cacheCluster.cacheClusterIdValidation()
        )`${Elasticache.sanitizeClusterName(lib.name, dbName)}`
        // create the db resources

        let redis
        if (elasticacheImports != undefined && elasticacheImports[dbName] != undefined) {
            redis = aws.elasticache.Cluster.get(clusterName, elasticacheImports[dbName])
        } else {
            redis = Elasticache.createCluster(lib, dbName, clusterName, args)
        }

        lib.topology.topologyIconData.forEach((resource) => {
            if (resource.kind == Resource.redis_node) {
                lib.topology.topologyEdgeData.forEach((edge) => {
                    var id = resource.id
                    if (edge.target == id && id == `${dbName}_${Resource.redis_node}`) {
                        lib.connectionString.set(
                            `${id}_host`,
                            pulumi.interpolate`${redis.cacheNodes.apply(
                                (nodes) => nodes[0].address
                            )}`
                        )
                        lib.connectionString.set(
                            `${id}_port`,
                            pulumi.interpolate`${redis.cacheNodes.apply((nodes) => nodes[0].port)}`
                        )

                        // Set another copy of the connection string for helm env vars in case they are used
                        const envVar = `${resource.title}${resource.kind}`
                            .toUpperCase()
                            .replace(/[^a-z0-9]/gi, '')
                        lib.connectionString.set(
                            `${envVar}HOST`,
                            pulumi.interpolate`${redis.cacheNodes.apply(
                                (nodes) => nodes[0].address
                            )}`
                        )
                        lib.connectionString.set(
                            `${envVar}PORT`,
                            pulumi.interpolate`${redis.cacheNodes.apply((nodes) => nodes[0].port)}`
                        )
                    }
                })
            }
        })
    }

    static sanitizeClusterName = (appName: string, dbName: string): string => {
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

    private static createCluster(
        lib: CloudCCLib,
        dbName: string,
        clusterName: string,
        args: Partial<aws.elasticache.ClusterArgs>
    ): aws.elasticache.Cluster {
        const logGroupName = `/aws/elasticache/${lib.name}-${dbName}-persist-redis`
        let cloudwatchGroup = new aws.cloudwatch.LogGroup(`persist-redis-${dbName}-lg`, {
            name: `${logGroupName}`,
            retentionInDays: 0,
        })

        const subnetGroup = new aws.elasticache.SubnetGroup(
            sanitized(
                AwsSanitizer.Elasticache.cacheSubnetGroup.cacheSubnetGroupNameValidation()
            )`${h(this.name)}-${h(dbName)}-subnetgroup`,
            {
                subnetIds: lib.privateSubnetIds,
                tags: {
                    Name: 'Klotho DB subnet group',
                },
            }
        )

        return new aws.elasticache.Cluster(
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
                subnetGroupName: subnetGroup.name,
                securityGroupIds: lib.sgs,
            },
            { protect: lib.protect }
        )
    }
}
