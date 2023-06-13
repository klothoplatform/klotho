import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Cluster: aws.docdb.Cluster
    InstanceClass: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.docdb.ClusterInstance {
    return new aws.docdb.ClusterInstance(args.Name, {
        clusterIdentifier: args.Cluster.clusterIdentifier,
        instanceClass: args.InstanceClass,
    })
}
