import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.ec2.Ami {
    return new aws.ec2.Ami(args.Name, {
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
