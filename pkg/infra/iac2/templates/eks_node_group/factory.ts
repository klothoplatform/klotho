import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Cluster: aws.eks.Cluster
    NodeRole: aws.iam.Role
    AmiType: string
    Subnets: aws.ec2.Subnet[]
    DesiredSize: number
    MinSize: number
    MaxSize: number
    MaxUnavailable: number
    DiskSize: number
    InstanceTypes: string[]
    Labels: Record<string, string>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.eks.NodeGroup {
    return new aws.eks.NodeGroup(args.Name, {
        clusterName: args.Cluster.name,
        nodeRoleArn: args.NodeRole.arn,
        //TMPL {{ if .AmiType.Raw }}
        amiType: args.AmiType,
        //TMPL {{ end }}
        subnetIds: args.Subnets.map((subnet) => subnet.id),
        scalingConfig: {
            desiredSize: args.DesiredSize,
            maxSize: args.MaxSize,
            minSize: args.MinSize,
        },
        updateConfig: {
            maxUnavailable: args.MaxUnavailable,
        },
        diskSize: args.DiskSize,
        instanceTypes: args.InstanceTypes,
        labels: args.Labels,
    })
}
