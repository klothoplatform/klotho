import * as aws from '@pulumi/aws'
import { TemplateWrapper, ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Id?: string
    Vpc: aws.ec2.Vpc
    Routes: TemplateWrapper<aws.types.input.ec2.RouteTableRoute[]>
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.RouteTable {
    return new aws.ec2.RouteTable(args.Name, {
        vpcId: args.Vpc.id,
        routes: args.Routes,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ec2.RouteTable, args: Args) {
    return {
        Id: object.id,
    }
}

function importResource(args: Args): aws.ec2.RouteTable {
    return aws.ec2.RouteTable.get(args.Name, args.Id)
}
