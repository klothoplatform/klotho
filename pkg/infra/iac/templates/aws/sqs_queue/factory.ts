import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    FifoQueue?: boolean
    DelaySeconds?: number
    MaxMessageSize?: number
    VisibilityTimeout?: number
    Tags: ModelCaseWrapper<Record<string, string>>
    protect: boolean
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.sqs.Queue {
    return new aws.sqs.Queue(
        args.Name,
        {
            //TMPL {{- if .FifoQueue }}
            fifoQueue: args.FifoQueue,
            //TMPL {{- end }}
            //TMPL {{- if .DelaySeconds }}
            delaySeconds: args.DelaySeconds,
            //TMPL {{- end }}
            //TMPL {{- if .MaxMessageSize }}
            maxMessageSize: args.MaxMessageSize,
            //TMPL {{- end }}
            //TMPL {{- if .VisibilityTimeout }}
            visibilityTimeoutSeconds: args.VisibilityTimeout,
            //TMPL {{- end }}
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        //TMPL {{- if .protect }}
        { protect: args.protect }
        //TMPL {{- end }}
    )
}

function properties(object: aws.sqs.Queue, args: Args) {
    return {
        Arn: object.arn,
    }
}

function importResource(args: Args): aws.sqs.Queue {
    return aws.sqs.Queue.get(args.Name, args.Id)
}
