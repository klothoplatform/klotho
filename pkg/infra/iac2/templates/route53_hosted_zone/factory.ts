import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpc: aws.ec2.Vpc
    ForceDestroy: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.route53.Zone {
    return new aws.route53.Zone(args.Name, {
        //TMPL {{- if .Vpc.Raw }}
        vpcs: [{ vpcId: args.Vpc.id }],
        //TMPL {{- end}}
        forceDestroy: args.ForceDestroy,
    })
}
