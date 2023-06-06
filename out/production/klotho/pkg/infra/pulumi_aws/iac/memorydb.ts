import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { Resource } from '../deploylib'
import { cacheCluster } from './sanitization/aws/memorydb'
import { CloudCCLib, kloConfig } from '../deploylib'
import { sanitized, hash as h } from './sanitization/sanitizer'
import AwsSanitizer from './sanitization/aws'

// Since not all zones are supported in us-east-1 and us-west-2 we will verify our subnets are valid for the subnet group
const supported_azs = ['use1-az2', 'use1-az4', 'use1-az6', 'usw2-az1', 'usw2-az2', 'usw2-az3']

export class MemoryDb {
    constructor() {}

    static async setupMemoryDbCluster(
        lib: CloudCCLib,
        dbName: string,
        args: Partial<aws.memorydb.ClusterArgs>
    ) {
        let subnets: string[] | Promise<pulumi.Output<string>[]> = []
        if (['us-east-1', 'us-west-2'].includes(lib.region)) {
            for (const subnetId in lib.privateSubnetIds) {
                const subnet: aws.ec2.GetSubnetResult = await aws.ec2.getSubnet({
                    id: subnetId,
                })
                if (supported_azs.includes(subnet.availabilityZoneId)) {
                    subnets.push(subnetId)
                }
            }
            if (subnets.length === 0) {
                throw new Error('Unable to find subnets in supported memorydb Availability Zones')
            }
        } else {
            subnets = lib.privateSubnetIds
        }

        const memoryDbImports: { [key: string]: string } | undefined =
            kloConfig.getObject('memorydb')
        // TODO: look into removing sanitizeClusterName when making other breaking changes to resource names
        const clusterName = sanitized(
            cacheCluster.clusterNameValidation()
        )`${MemoryDb.sanitizeClusterName(lib.name, dbName)}`
        // create the db resources

        let memdbCluster
        if (memoryDbImports != undefined && memoryDbImports[dbName] != undefined) {
            memdbCluster = aws.memorydb.Cluster.get(clusterName, memoryDbImports[dbName])
        } else {
            memdbCluster = MemoryDb.createCluster(lib, dbName, clusterName, subnets, args)
        }

        lib.topology.topologyIconData.forEach((resource) => {
            if (resource.kind == Resource.redis_cluster) {
                lib.topology.topologyEdgeData.forEach((edge) => {
                    var id = resource.id
                    if (edge.target == id && id == `${dbName}_${Resource.redis_cluster}`) {
                        lib.connectionString.set(
                            `${id}_host`,
                            pulumi.interpolate`${memdbCluster.clusterEndpoints.apply(
                                (endpoint) => endpoint[0].address
                            )}`
                        )
                        lib.connectionString.set(
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

    static sanitizeClusterName(appName: string, dbName: string): string {
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
        subnets: string[] | Promise<pulumi.Output<string>[]>,
        args: Partial<aws.memorydb.ClusterArgs>
    ): aws.memorydb.Cluster {
        const subnetGroup = new aws.memorydb.SubnetGroup(
            sanitized(AwsSanitizer.MemoryDB.subnetGroup.subnetGroupNameValidation())`${
                lib.name
            }-${h(dbName)}-subnetgroup`,
            {
                subnetIds: subnets,
                tags: {
                    Name: 'Klotho DB subnet group',
                },
            }
        )

        return new aws.memorydb.Cluster(
            clusterName,
            {
                ...args,
                name: clusterName,
                aclName: 'open-access',
                nodeType: 'db.t4g.small',
                securityGroupIds: lib.sgs,
                snapshotRetentionLimit: 7,
                subnetGroupName: subnetGroup.name,
            },
            { protect: lib.protect }
        )
    }
}
