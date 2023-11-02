import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    LogGroupName: string
    RetentionInDays: number
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.cloudwatch.LogGroup {
    return new aws.cloudwatch.LogGroup(args.Name, {
        name: args.LogGroupName,
        retentionInDays: args.RetentionInDays,
    })
}

function properties(object: aws.cloudwatch.LogGroup, args: Args) {
    return {
        Arn: object.arn,
    }
}
