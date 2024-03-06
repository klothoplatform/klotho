import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.elasticache.SubnetGroup {
    return new aws.elasticache.SubnetGroup(args.Name, {
        subnetIds: args.Subnets.map((sg) => sg.id),
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
