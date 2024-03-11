import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Description: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.servicediscovery.HttpNamespace {
    return new aws.servicediscovery.HttpNamespace(args.Name, {
        //TMPL {{- if .Description }}
        description: args.Description,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.servicediscovery.HttpNamespace, args: Args) {
    return {
        Id: object.id,
        Arn: object.arn,
    }
}
