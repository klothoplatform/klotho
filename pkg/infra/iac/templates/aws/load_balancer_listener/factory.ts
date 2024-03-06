import * as aws from '@pulumi/aws'
import { TemplateWrapper, ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Port: number
    Protocol: string
    LoadBalancer: aws.lb.LoadBalancer
    DefaultActions: TemplateWrapper<aws.types.input.lb.ListenerDefaultAction[]>
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.Listener {
    return new aws.lb.Listener(args.Name, {
        loadBalancerArn: args.LoadBalancer.arn,
        defaultActions: args.DefaultActions,
        port: args.Port,
        protocol: args.Protocol,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
