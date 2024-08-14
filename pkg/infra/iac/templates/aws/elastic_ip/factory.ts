import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Id?: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Eip {
    return new aws.ec2.Eip(args.Name, {
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.Eip, args: Args) {
    return {
        Id: object.id,
    }
}

function importResource(args: Args): aws.ec2.Eip {
    return aws.ec2.Eip.get(args.Name, args.Id)
}
