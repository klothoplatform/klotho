import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Port: number
    Protocol: string
    Vpc: aws.ec2.Vpc
    TargetType: string
    Tags: Record<string, string>
    Targets: { Id: string; Port: number }[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.TargetGroup {
    return (() => {
        const tg = new aws.lb.TargetGroup(args.Name, {
            port: args.Port,
            protocol: args.Protocol,
            targetType: args.TargetType,
            vpcId: args.Vpc.id,
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        })

        //TMPL {{- if .Targets }}
        for (const target of args.Targets) {
            new aws.lb.TargetGroupAttachment(target.Id, {
                port: target.Port,
                targetGroupArn: tg.arn,
                targetId: target.Id,
            })
        }
        //TMPL {{- end }}
        return tg
    })()
}
