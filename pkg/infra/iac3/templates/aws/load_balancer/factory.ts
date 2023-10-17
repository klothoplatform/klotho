import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    IpAddressType: string
    LoadBalancerAttributes: Record<string, string>
    Scheme: string
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnets: aws.ec2.Subnet[]
    Tags: Record<string, string>
    Type: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.LoadBalancer {
    return new aws.lb.LoadBalancer(args.Name, {
        internal: args.Scheme == 'internal',
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
        NlbUri: pulumi.interpolate`http://${object.dnsName}`
    }
}