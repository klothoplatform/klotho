import { local } from '@pulumi/command'
import * as pulumi from '@pulumi/pulumi'
import * as aws from '@pulumi/aws'
import { Resource, TopologyData } from '../deploylib'

const config = new pulumi.Config()

export interface Options {
    topology: TopologyData
    app: string
    region: string
    id: string
    spendLimit?: number
}

export class CockroachDB extends pulumi.ComponentResource {
    public readonly connectionString: pulumi.Output<string>

    constructor(name: string, args: Options, opts?: pulumi.ComponentResourceOptions) {
        super('klotho:cockroachdb:ServerlessDB', name, args, opts)

        const cluster = this.clusterName(args)
        const spendLimit = args.spendLimit ? `--spend-limit ${args.spendLimit}` : ''

        const clusterRes = new local.Command(
            `${cluster}-cluster`,
            {
                create: `ccloud cluster create serverless --cloud 'AWS' --quiet ${spendLimit} ${cluster} ${args.region}`,
                update:
                    spendLimit.length > 0
                        ? `ccloud cluster update --quiet ${spendLimit} ${cluster}`
                        : 'echo no-op',
                delete: `ccloud cluster delete --quiet ${cluster}`,
            },
            { parent: this }
        )

        const username = config.require(`${args.id}_username`)
        const password = config.requireSecret(`${args.id}_password`)

        const user = new local.Command(
            `${args.id}-user`,
            {
                create: pulumi.interpolate`ccloud cluster user create --quiet ${cluster} ${username} -p '${password}'`,
                update: 'echo no-op',
            },
            { dependsOn: [clusterRes], parent: clusterRes }
        )

        const connectionString = new local.Command(
            `${args.id}-conn`,
            {
                create: `ccloud cluster sql --quiet --connection-url --cert-path 'root.crt' ${cluster}`,
            },
            { dependsOn: [clusterRes, user], parent: clusterRes }
        )

        const copyCerts: local.Command[] = []
        const resourceId = `${args.id}_${Resource.orm}`
        for (const edge of args.topology.topologyEdgeData) {
            if (edge.target != resourceId) {
                continue
            }
            const source = args.topology.topologyIconData.find((elem) => elem.id == edge.source)!

            copyCerts.push(
                new local.Command(
                    `${args.id}-cert-${source.title}`,
                    {
                        create: `cp root.crt ${source.title}/`,
                    },
                    { dependsOn: [connectionString], parent: connectionString }
                )
            )
        }

        this.connectionString = pulumi
            .all([connectionString.stdout, password, ...copyCerts.map((c) => c.stdout)])
            .apply(([s, p]) => s.replace('postgresql://', `postgresql://${username}:${p}@`))

        this.registerOutputs({
            connectionString: this.connectionString,
        })
    }

    private clusterName(opts: Options): string {
        let cluster = `${opts.app}-${opts.id}`.toLowerCase()
        if (cluster.length >= 20) {
            const oldCluster = cluster
            cluster = ''
            if (opts.app.length > 6) {
                cluster += opts.app.substring(0, 3) + opts.app.substring(opts.app.length - 3)
            } else {
                cluster += opts.app
            }
            cluster += '-'
            if (opts.id.length > 12) {
                cluster += opts.id.substring(0, 6) + opts.id.substring(opts.id.length - 6)
            } else {
                cluster += opts.id
            }
            cluster = cluster.toLowerCase()

            console.info(
                `Abbreviating cluster name '${oldCluster}' to '${cluster}' due to exceeding length limits (${oldCluster.length} >= 20)`
            )
        }
        return cluster
    }
}
