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

function properties(object: aws.rds.SubnetGroup, args: Args) {
    return {
        DeployedName: object.name,
    }
}

type AllProperties = Args & ReturnType<typeof properties>

function importResource(args: AllProperties): aws.rds.SubnetGroup {
    return aws.rds.SubnetGroup.get(args.Name, args.DeployedName)
}