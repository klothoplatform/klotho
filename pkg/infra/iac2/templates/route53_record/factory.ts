import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
interface Args {
    Name: string
    DomainName: string
    Zone: aws.route53.Zone
    Type: string
    Records: pulumi.Output<string>[]
    HealthCheck: aws.route53.HealthCheck
    TTL: number
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.route53.Record {
    return new aws.route53.Record(args.Name, {
        //TMPL {{- if .HealthCheck.Raw }}
        healthCheckId: args.HealthCheck.id,
        //TMPL {{- end}}
        zoneId: args.Zone.id,
        type: args.Type,
        records: args.Records,
        ttl: args.TTL,
        name: args.DomainName,
    })
}
