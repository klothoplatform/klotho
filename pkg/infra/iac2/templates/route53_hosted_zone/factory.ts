import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Vpcs: aws.ec2.Vpc[]
    ForceDestroy: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.route53.Zone {
    return new aws.route53.Zone(args.Name, {
        //TMPL {{- if .Vpc.Raw }}
        vpcs: args.Vpcs.map((vpc) => {
            return { vpcId: vpc.id }
        }),
        //TMPL {{- end}}
        forceDestroy: args.ForceDestroy,
    })
}
