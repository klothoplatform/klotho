import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    protect: boolean
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.secretsmanager.Secret {
    return new aws.secretsmanager.Secret(
        args.Name,
        {
            name: args.Name,
            recoveryWindowInDays: 0,
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        { protect: args.protect }
    )
}

function properties(object: aws.secretsmanager.Secret, args: Args) {
    return {
        Arn: object.arn,
        Id: object.id,
    }
}
