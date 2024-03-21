import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    ApplicationFailureFeedbackRoleArn: string
    ApplicationSuccessFeedbackRoleArn: string
    ApplicationSuccessFeedbackSampleRate: number
    ArchivePolicy: string
    ContentBasedDeduplication: boolean
    DeliveryPolicy: string
    FifoTopic: boolean
    FirehoseFailureFeedbackRoleArn: string
    FirehoseSuccessFeedbackRoleArn: string
    FirehoseSuccessFeedbackSampleRate: number
    HttpFailureFeedbackRoleArn: string
    HttpSuccessFeedbackRoleArn: string
    HttpSuccessFeedbackSampleRate: number
    KmsMasterKeyId: string
    LambdaFailureFeedbackRoleArn: string
    LambdaSuccessFeedbackRoleArn: string
    LambdaSuccessFeedbackSampleRate: number
    Policy: string
    SignatureVersion: number
    SqsFailureFeedbackRoleArn: string
    SqsSuccessFeedbackRoleArn: string
    SqsSuccessFeedbackSampleRate: number
    TracingConfig: string
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.sns.Topic {
    return new aws.sns.Topic(
        args.Name,
        {
            //TMPL {{- if .ApplicationFailureFeedbackRoleArn }}
            applicationFailureFeedbackRoleArn: args.ApplicationFailureFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .ApplicationSuccessFeedbackRoleArn }}
            applicationSuccessFeedbackRoleArn: args.ApplicationSuccessFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .ApplicationSuccessFeedbackSampleRate }}
            applicationSuccessFeedbackSampleRate: args.ApplicationSuccessFeedbackSampleRate,
            //TMPL {{- end }}
            //TMPL {{- if .ArchivePolicy }}
            archivePolicy: args.ArchivePolicy,
            //TMPL {{- end }}
            //TMPL {{- if .ContentBasedDeduplication }}
            contentBasedDeduplication: args.ContentBasedDeduplication,
            //TMPL {{- end }}
            //TMPL {{- if .DeliveryPolicy }}
            deliveryPolicy: args.DeliveryPolicy,
            //TMPL {{- end }}
            //TMPL {{- if .FifoTopic }}
            fifoTopic: args.FifoTopic,
            //TMPL {{- end }}
            //TMPL {{- if .FirehoseFailureFeedbackRoleArn }}
            firehoseFailureFeedbackRoleArn: args.FirehoseFailureFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .FirehoseSuccessFeedbackRoleArn }}
            firehoseSuccessFeedbackRoleArn: args.FirehoseSuccessFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .FirehoseSuccessFeedbackSampleRate }}
            firehoseSuccessFeedbackSampleRate: args.FirehoseSuccessFeedbackSampleRate,
            //TMPL {{- end }}
            //TMPL {{- if .HttpFailureFeedbackRoleArn }}
            httpFailureFeedbackRoleArn: args.HttpFailureFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .HttpSuccessFeedbackRoleArn }}
            httpSuccessFeedbackRoleArn: args.HttpSuccessFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .HttpSuccessFeedbackSampleRate }}
            httpSuccessFeedbackSampleRate: args.HttpSuccessFeedbackSampleRate,
            //TMPL {{- end }}
            //TMPL {{- if .KmsMasterKeyId }}
            kmsMasterKeyId: args.KmsMasterKeyId,
            //TMPL {{- end }}
            //TMPL {{- if .LambdaFailureFeedbackRoleArn }}
            lambdaFailureFeedbackRoleArn: args.LambdaFailureFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .LambdaSuccessFeedbackRoleArn }}
            lambdaSuccessFeedbackRoleArn: args.LambdaSuccessFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .LambdaSuccessFeedbackSampleRate }}
            lambdaSuccessFeedbackSampleRate: args.LambdaSuccessFeedbackSampleRate,
            //TMPL {{- end }}
            //TMPL {{- if .Policy }}
            policy: args.Policy,
            //TMPL {{- end }}
            //TMPL {{- if .SignatureVersion }}
            signatureVersion: args.SignatureVersion,
            //TMPL {{- end }}
            //TMPL {{- if .SqsFailureFeedbackRoleArn }}
            sqsFailureFeedbackRoleArn: args.SqsFailureFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .SqsSuccessFeedbackRoleArn }}
            sqsSuccessFeedbackRoleArn: args.SqsSuccessFeedbackRoleArn,
            //TMPL {{- end }}
            //TMPL {{- if .SqsSuccessFeedbackSampleRate }}
            sqsSuccessFeedbackSampleRate: args.SqsSuccessFeedbackSampleRate,
            //TMPL {{- end }}
            //TMPL {{- if .TracingConfig }}
            tracingConfig: args.TracingConfig,
            //TMPL {{- end }}
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
    )
}

function properties(object: aws.sns.Topic, args: Args) {
    return {
        Arn: object.arn,
        ID: object.id,
    }
}

function importResource(args: Args): aws.sns.Topic {
    return aws.sns.Topic.get(args.Name, args.Id)
}
