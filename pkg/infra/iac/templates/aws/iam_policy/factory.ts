import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Policy: ModelCaseWrapper<aws.iam.PolicyDocument>
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.iam.Policy {
    return new aws.iam.Policy(args.Name, {
        policy: args.Policy,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.iam.Policy, args: Args) {
    return {
        Arn: object.arn,
    }
}
