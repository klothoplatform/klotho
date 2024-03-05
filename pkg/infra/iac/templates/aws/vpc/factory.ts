import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    CidrBlock: string
    EnableDnsHostnames: boolean
    EnableDnsSupport: boolean
    Arn?: string
    Id?: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.Vpc {
    return new aws.ec2.Vpc(args.Name, {
        cidrBlock: args.CidrBlock,
        enableDnsHostnames: args.EnableDnsHostnames,
        enableDnsSupport: args.EnableDnsSupport,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.Vpc, args: Args) {
    return {
        Id: object.id,
        Arn: object.arn,
    }
}

function importResource(args: Args): aws.ec2.Vpc {
    return aws.ec2.Vpc.get(args.Name, args.Id)
}
