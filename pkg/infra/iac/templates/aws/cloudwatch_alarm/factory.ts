import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    ComparisonOperator: string
    EvaluationPeriods: number
    ActionsEnabled: boolean
    AlarmActions: string[]
    AlarmDescription: string
    DatapointsToAlarm: string
    Dimensions: ModelCaseWrapper<Record<string, string>>
    ExtendedStatistic: string
    InsufficientDataActions: string[]
    MetricName: string
    Namespace: string
    OKActions: string[]
    Period: number
    Statistic: string
    Threshold: number
    TreatMissingData: string
    Unit: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.cloudwatch.MetricAlarm {
    return new aws.cloudwatch.MetricAlarm(args.Name, {
        //TMPL {{- if .ComparisonOperator }}
        comparisonOperator: args.ComparisonOperator,
        //TMPL {{- end }}
        //TMPL {{- if .EvaluationPeriods }}
        evaluationPeriods: args.EvaluationPeriods,
        //TMPL {{- end }}
        //TMPL {{- if .ActionsEnabled }}
        actionsEnabled: args.ActionsEnabled,
        //TMPL {{- end }}
        //TMPL {{- if .AlarmActions }}
        alarmActions: args.AlarmActions,
        //TMPL {{- end }}
        //TMPL {{- if .AlarmDescription }}
        alarmDescription: args.AlarmDescription,
        //TMPL {{- end }}
        //TMPL {{- if .DatapointsToAlarm }}
        datapointsToAlarm: args.DatapointsToAlarm,
        //TMPL {{- end }}
        //TMPL {{- if .Dimensions }}
        dimensions: args.Dimensions,
        //TMPL {{- end }}
        //TMPL {{- if .ExtendedStatistic }}
        extendedStatistic: args.ExtendedStatistic,
        //TMPL {{- end }}
        //TMPL {{- if .InsufficientDataActions }}
        insufficientDataActions: args.InsufficientDataActions,
        //TMPL {{- end }}
        //TMPL {{- if .MetricName }}
        metricName: args.MetricName,
        //TMPL {{- end }}
        //TMPL {{- if .Namespace }}
        namespace: args.Namespace,
        //TMPL {{- end }}
        //TMPL {{- if .OKActions }}
        okActions: args.OKActions,
        //TMPL {{- end }}
        //TMPL {{- if .Period }}
        period: args.Period,
        //TMPL {{- end }}
        //TMPL {{- if .Statistic }}
        statistic: args.Statistic,
        //TMPL {{- end }}
        //TMPL {{- if .Threshold }}
        threshold: args.Threshold,
        //TMPL {{- end }}
        //TMPL {{- if .TreatMissingData }}
        treatMissingData: args.TreatMissingData,
        //TMPL {{- end }}
        //TMPL {{- if .Unit }}
        unit: args.Unit,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.cloudwatch.MetricAlarm, args: Args) {
    return {
        Arn: object.arn,
    }
}
