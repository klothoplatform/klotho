import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    EventSource: aws.sqs.Queue
    Function: aws.lambda.Function
    FilterCriteria?: ModelCaseWrapper<Record<string, string>[]>
    BatchSize?: number
    Enabled?: boolean
    FunctionResponseTypes?: string[]
    MaximumBatchingWindowInSeconds?: number
    ScalingConfig?: object
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.lambda.EventSourceMapping {
    return new aws.lambda.EventSourceMapping(
        args.Name,
        {
            eventSourceArn: args.EventSource.arn,
            functionName: args.Function.name,
            //TMPL {{- if .FilterCriteria }}
            filterCriteria: {
                filters: args.FilterCriteria,
            },
            //TMPL {{- end }}
            //TMPL {{- if .BatchSize }}
            batchSize: args.BatchSize,
            //TMPL {{- end }}
            //TMPL {{- if .Enabled }}
            enabled: args.Enabled,
            //TMPL {{- end }}
            //TMPL {{- if .FunctionResponseTypes }}
            functionResponseTypes: args.FunctionResponseTypes,
            //TMPL {{- end }}
            //TMPL {{- if .MaximumBatchingWindowInSeconds }}
            maximumBatchingWindowInSeconds: args.MaximumBatchingWindowInSeconds,
            //TMPL {{- end }}
            //TMPL {{- if .ScalingConfig }}
            scalingConfig: args.ScalingConfig,
            //TMPL {{- end }}
        },
        {
            dependsOn: args.dependsOn,
        }
    )
}
