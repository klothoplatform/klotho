import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    SecurityGroups: aws.ec2.SecurityGroup[]
    ClusterRole: aws.iam.Role
    Version: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.eks.Cluster {
    return new aws.eks.Cluster(args.Name, {
        version: args.Version,
        vpcConfig: {
            subnetIds: args.Subnets.map((subnet) => subnet.id),
            //TMPL {{- if .SecurityGroups }}
            securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
            //TMPL {{- end }}
        },
        roleArn: args.ClusterRole.arn,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.eks.Cluster, args: Args) {
    return {
        Name: object.name,
        ClusterEndpoint: object.endpoint,
        CertificateAuthorityData: object.certificateAuthorities[0].data,
        ClusterSecurityGroup: object.vpcConfig.clusterSecurityGroupId,
    }
}
