import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.servicediscovery.PrivateDnsNamespace {
    return new aws.servicediscovery.PrivateDnsNamespace(args.Name, {
        vpc: args.Vpc.id,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.servicediscovery.PrivateDnsNamespace, args: Args) {
    return {
        Id: object.id,
        Name: object.name,
    }
}
