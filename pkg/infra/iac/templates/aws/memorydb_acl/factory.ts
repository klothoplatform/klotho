import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    UserNames: string[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.memorydb.Acl {
    return new aws.memorydb.Acl(args.Name, {
        //TMPL {{- if .UserNames }}
        userNames: args.UserNames,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.memorydb.Acl, args: Args) {
    return {
        Arn: object.arn,
        Id: object.id,
    }
}
