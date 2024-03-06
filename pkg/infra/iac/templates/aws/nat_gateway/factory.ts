import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    ElasticIp: aws.ec2.Eip
    Subnet: aws.ec2.Subnet
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ec2.NatGateway {
    return new aws.ec2.NatGateway(args.Name, {
        allocationId: args.ElasticIp.id,
        subnetId: args.Subnet.id,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
