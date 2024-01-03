import * as aws from '@pulumi/aws'
import * as inputs from '@pulumi/aws/types/input'
import { ModelCaseWrapper, TemplateWrapper } from '../../wrappers'

interface Args {
    Name: string
    Listener: aws.lb.Listener
    Priority: number
    Conditions: []
    Actions: TemplateWrapper<inputs.lb.ListenerRuleAction[]>
    Tags?: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lb.ListenerRule {
    return new aws.lb.ListenerRule(args.Name, {
        listenerArn: args.Listener.arn,
        priority: args.Priority,
        conditions: args.Conditions,
        actions: args.Actions,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}
