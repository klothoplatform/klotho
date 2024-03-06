import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Engine: string
    CloudwatchGroup: aws.cloudwatch.LogGroup
    SubnetGroup: aws.elasticache.SubnetGroup
    SecurityGroups: aws.ec2.SecurityGroup[]
    NodeType: string
    NumCacheNodes: number
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.elasticache.Cluster {
    return new aws.elasticache.Cluster(args.Name, {
        engine: args.Engine,
        nodeType: args.NodeType,
        numCacheNodes: args.NumCacheNodes,
        logDeliveryConfigurations: [
            {
                destination: args.CloudwatchGroup.name,
                destinationType: 'cloudwatch-logs',
                logFormat: 'text',
                logType: 'slow-log',
            },
            {
                destination: args.CloudwatchGroup.name,
                destinationType: 'cloudwatch-logs',
                logFormat: 'json',
                logType: 'engine-log',
            },
        ],
        subnetGroupName: args.SubnetGroup.name,
        securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.elasticache.Cluster, args: Args) {
    return {
        Port: object.port,
        ClusterAddress: object.clusterAddress,
        CacheNodeAddress: object.cacheNodes.apply((nodes) => nodes[0].address),
    }
}
