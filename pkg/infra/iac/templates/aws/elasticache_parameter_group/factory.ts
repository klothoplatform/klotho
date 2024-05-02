import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Description: string
    Family: string
    Parameters: Record<string, string>
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.elasticache.SubnetGroup {
    return new aws.elasticache.ParameterGroup(args.Name, {
        name: args.Name,
        family: args.Family,
        description: args.Description,
        //TMPL {{- if .Parameters }}
        parameters: args.Parameters,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.elasticache.ParameterGroup, args: Args) {
    return {
        Arn: object.arn,
        Name: object.name,
    }
}
