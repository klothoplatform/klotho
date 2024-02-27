import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.rds.SubnetGroup {
    return new aws.rds.SubnetGroup(args.Name, {
        subnetIds: args.Subnets.map((subnet) => subnet.id),
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
