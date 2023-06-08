import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Port: number
    TargetGroupArn: string
    TargetId: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.TargetGroupAttachment {
    return new aws.lb.TargetGroupAttachment(args.Name, {
        port: args.Port,
        targetGroupArn: args.TargetGroupArn,
        targetId: args.TargetId,
    })
}
