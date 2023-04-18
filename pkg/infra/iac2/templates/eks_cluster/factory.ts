import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    SecurityGroups: aws.ec2.SecurityGroup[]
    ClusterRole: aws.iam.Role
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.eks.Cluster {
    return new aws.eks.Cluster(args.Name, {
        vpcConfig: {
            subnetIds: args.Subnets.map((subnet) => subnet.id),
            //TMPL {{- if .SecurityGroups.Raw }}
            securityGroupIds: args.SecurityGroups.map((sg) => sg.id),
            //TMPL {{- end }}
        },
        roleArn: args.ClusterRole.arn,
    })
}
