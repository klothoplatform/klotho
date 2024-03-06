import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    IngressRules: aws.types.input.ec2.SecurityGroupIngress[]
    EgressRules: aws.types.input.ec2.SecurityGroupEgress[]
    Tags: ModelCaseWrapper<Record<string, string>>
}

function create(args: Args): aws.ec2.SecurityGroup {
    return new aws.ec2.SecurityGroup(args.Name, {
        name: args.Name,
        vpcId: args.Vpc.id,
        egress: args.EgressRules,
        ingress: args.IngressRules,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
