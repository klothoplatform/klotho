import * as aws from '@pulumi/aws'
import * as awsInputs from '@pulumi/aws/types/input'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Subnets: aws.ec2.Subnet[]
    Description: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.memorydb.SubnetGroup {
    return new aws.memorydb.SubnetGroup(args.Name, {
        subnetIds: args.Subnets.map((subnet) => subnet.id),
        //TMPL {{- if .Description }}
        description: args.Description,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.memorydb.SubnetGroup, args: Args) {
    return {
        Arn: object.arn,
        Id: object.id,
    }
}
