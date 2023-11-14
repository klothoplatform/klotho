import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    FileSystem: aws.efs.FileSystem
    IpAddress?: string
    SecurityGroups?: aws.ec2.SecurityGroup[]
    Subnet: aws.ec2.Subnet
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.efs.MountTarget {
    return new aws.efs.MountTarget(args.Name, {
        fileSystemId: args.FileSystem.id,
        //TMPL {{- if .IpAddress }}
        ipAddress: args.IpAddress,
        //TMPL {{- end }}
        subnetId: args.Subnet.id,
        //TMPL {{- if .SecurityGroups }}
        securityGroups: args.SecurityGroups?.map((sg) => sg.id),
        //TMPL {{- end }}
    })
}
