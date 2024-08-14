import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    LogGroupName: string
    RetentionInDays: number
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.cloudwatch.LogGroup {
    return new aws.cloudwatch.LogGroup(args.Name, {
        //TMPL {{- if .LogGroupName }}
        name: args.LogGroupName,
        //TMPL {{- end }}
        retentionInDays: args.RetentionInDays,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.cloudwatch.LogGroup, args: Args) {
    return {
        Arn: object.arn,
    }
}
