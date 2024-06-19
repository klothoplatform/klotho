import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Id?: string
    Vpc: aws.ec2.Vpc
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.InternetGateway {
    return new aws.ec2.InternetGateway(args.Name, {
        vpcId: args.Vpc.id,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.InternetGateway, args: Args) {
    return {
        Id: object.id,
    }
}

function importResource(args: Args): aws.ec2.InternetGateway {
    return aws.ec2.InternetGateway.get(args.Name, args.Id)
}
