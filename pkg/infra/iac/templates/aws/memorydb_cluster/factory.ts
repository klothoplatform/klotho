import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    Acl: string
    NodeType: string
    AutoMinorVersionUpgrade: boolean
    DataTiering: string
    Description: string
    EngineVersion: string
    FinalSnapshotName: string
    MaintenanceWindow: string
    NumReplicasPerShard: number
    NumShards: number
    ParameterGroupName: string
    Port: number
    SecurityGroups: aws.ec2.SecurityGroup[]
    SnapshotArns: string[]
    SnapshotName: string
    SnapshotRetentionLimit: number
    SnapshotWindow: string
    SubnetGroup: aws.memorydb.SubnetGroup
    Tags: ModelCaseWrapper<Record<string, string>>
    TlsEnabled: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.memorydb.Cluster {
    return new aws.memorydb.Cluster(args.Name, {
        aclName: args.Acl,
        nodeType: args.NodeType,
        //TMPL {{- if .AutoMinorVersionUpgrade }}
        autoMinorVersionUpgrade: args.AutoMinorVersionUpgrade,
        //TMPL {{- end }}
        //TMPL {{- if .DataTiering }}
        dataTiering: args.DataTiering,
        //TMPL {{- end }}
        //TMPL {{- if .Description }}
        description: args.Description,
        //TMPL {{- end }}
        //TMPL {{- if .EngineVersion }}
        engineVersion: args.EngineVersion,
        //TMPL {{- end }}
        //TMPL {{- if .FinalSnapshotName }}
        finalSnapshotName: args.FinalSnapshotName,
        //TMPL {{- end }}
        //TMPL {{- if .MaintenanceWindow }}
        maintenanceWindow: args.MaintenanceWindow,
        //TMPL {{- end }}
        //TMPL {{- if .NumReplicasPerShard }}
        numReplicasPerShard: args.NumReplicasPerShard,
        //TMPL {{- end }}
        //TMPL {{- if .NumShards }}
        numShards: args.NumShards,
        //TMPL {{- end }}
        //TMPL {{- if .ParameterGroupName }}
        parameterGroupName: args.ParameterGroupName,
        //TMPL {{- end }}
        //TMPL {{- if .Port }}
        port: args.Port,
        //TMPL {{- end }}
        //TMPL {{- if .SecurityGroups }}
        securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
        //TMPL {{- end }}
        //TMPL {{- if .SnapshotArns }}
        snapshotArns: args.SnapshotArns,
        //TMPL {{- end }}
        //TMPL {{- if .SnapshotName }}
        snapshotName: args.SnapshotName,
        //TMPL {{- end }}
        //TMPL {{- if .SnapshotRetentionLimit }}
        snapshotRetentionLimit: args.SnapshotRetentionLimit,
        //TMPL {{- end }}
        //TMPL {{- if .SnapshotWindow }}
        snapshotWindow: args.SnapshotWindow,
        //TMPL {{- end }}
        //TMPL {{- if .SubnetGroup }}
        subnetGroupName: args.SubnetGroup.name,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
        //TMPL {{- if .TlsEnabled }}
        tlsEnabled: args.TlsEnabled,
        //TMPL {{- end }}
    })
}

function properties(object: aws.memorydb.Cluster, args: Args) {
    return {
        Arn: object.arn,
        ClusterEndpoints: object.clusterEndpoints,
        ClusterEndpointString: pulumi.interpolate`${object.clusterEndpoints.apply((endpoints) => {
            const endpointStrings: string[] = []
            for (const endpoint of endpoints) {
                endpointStrings.push(`${endpoint.address}:${endpoint.port}`)
            }
            return endpointStrings.join(',')
        })}`,
    }
}
