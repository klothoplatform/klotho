import * as aws from '@pulumi/aws'

interface Args {
    Name: string
    Type: string
    Disabled: boolean
    FailureThreshold: number
    Fqdn: string
    IpAddress: string
    Port: number
    RequestInterval: number
    ResourcePath: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.route53.HealthCheck {
    return new aws.route53.HealthCheck(args.Name, {
        type: args.Type,
        //TMPL {{- if .Fqdn.Raw }}
        fqdn: args.Fqdn,
        //TMPL {{- end}}
        //TMPL {{- if .IpAddress.Raw }}
        ipAddress: args.IpAddress,
        //TMPL {{- end}}
        disabled: args.Disabled,
        failureThreshold: args.FailureThreshold,
        port: args.Port,
        requestInterval: args.RequestInterval,
        resourcePath: args.ResourcePath,
    })
}
