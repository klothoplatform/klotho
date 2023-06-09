import * as aws from '@pulumi/aws'
import * as pulumi from "@pulumi/pulumi";

interface Args {
    BaseName: string
    Cluster: aws.docdb.Cluster
    InstanceClass: string
    Count: number
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.docdb.ClusterInstance[] {
    return ".".repeat(5).split('').map((_, idx) => {
        return new aws.docdb.ClusterInstance(args.BaseName + `-${idx}`, {
            clusterIdentifier: args.Cluster.clusterIdentifier,
            instanceClass: args.InstanceClass,
        })
    });
}
