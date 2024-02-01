import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'

interface Args {
    Name: string
    DnsConfig: aws.servicediscovery.ServiceDnsConfig
    HealthCheckCustomConfig: aws.servicediscovery.ServiceHealthCheckCustomConfig
}

function create(args: Args): aws.servicediscovery.Service {
    return new aws.servicediscovery.Service(args.Name, {
        dnsConfig: args.DnsConfig,
        //TMPL {{- if .HealthCheckCustomConfig }}
        healthCheckCustomConfig: args.HealthCheckCustomConfig,
        //TMPL {{- end }}
    })
}

function properties(object: aws.servicediscovery.Service, args: Args) {
    return {
        Arn: object.arn,
        Name: object.name,
    }
}
