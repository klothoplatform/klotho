import * as aws from '@pulumi/aws'

interface Args {
    SanitizedName: string
    Port: number
    Protocol: string
    Vpc: aws.ec2.Vpc
    TargetType: string
    Tags: Record<string, string>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.TargetGroup {
    return new aws.lb.TargetGroup(args.SanitizedName, {
        port: args.Port,
        protocol: args.Protocol,
        targetType: args.TargetType,
        vpcId: args.Vpc.id,
        //TMPL {{- if .Tags.Raw }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
