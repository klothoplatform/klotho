import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    IpAddressType: string
    LoadBalancerAttributes: Record<string, string>
    Scheme: string
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnets: aws.ec2.Subnet[]
    Tags: ModelCaseWrapper<Record<string, string>>
    Type: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.LoadBalancer {
    return new aws.lb.LoadBalancer(args.Name, {
        //TMPL {{- if eq .Scheme "internal" }}
        internal: true,
        //TMPL {{- else }}
        internal: false,
        //TMPL {{- end }}
        loadBalancerType: args.Type,
        subnets: args.Subnets.map((subnet) => subnet.id),
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
        //TMPL {{- if .SecurityGroups }}
        securityGroups: args.SecurityGroups.map((sg) => sg.id),
        //TMPL {{- end }}
    })
}

function properties(object: aws.lb.LoadBalancer, args: Args) {
    return {
        NlbUri: pulumi.interpolate`http://${object.dnsName}`,
    }
}

function infraExports(
    object: ReturnType<typeof create>,
    args: Args,
    props: ReturnType<typeof properties>
) {
    return {
        DomainName: object.dnsName,
    }
}
