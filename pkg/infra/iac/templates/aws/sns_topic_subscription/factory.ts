import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'

interface Args {
    Name: string
    Endpoint: string
    Protocol: string
    Topic: string
    ConfirmationTimeoutInMinutes: number
    DeliveryPolicy: string
    EndpointAutoConfirms: boolean
    FilterPolicy: string
    FilterPolicyScope: string
    RawMessageDelivery: boolean
    RedrivePolicy: string
    ReplayPolicy: string
    SubscriptionRoleArn: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.sns.TopicSubscription {
    return new aws.sns.TopicSubscription(
        args.Name,
        {
            endpoint: args.Endpoint,
            protocol: args.Protocol,
            topic: args.Topic,
            //TMPL {{- if .ConfirmationTimeoutInMinutes }}
            confirmationTimeoutInMinutes: args.ConfirmationTimeoutInMinutes,
            //TMPL {{- end }}
            //TMPL {{- if .DeliveryPolicy }}
            deliveryPolicy: args.DeliveryPolicy,
            //TMPL {{- end }}
            //TMPL {{- if .EndpointAutoConfirms }}
            endpointAutoConfirms: args.EndpointAutoConfirms,
            //TMPL {{- end }}
            //TMPL {{- if .FilterPolicy }}
            filterPolicy: args.FilterPolicy,
            //TMPL {{- end }}
            //TMPL {{- if .FilterPolicyScope }}
            filterPolicyScope: args.FilterPolicyScope,
            //TMPL {{- end }}
            //TMPL {{- if .RawMessageDelivery }}
            rawMessageDelivery: args.RawMessageDelivery,
            //TMPL {{- end }}
            //TMPL {{- if .RedrivePolicy }}
            redrivePolicy: args.RedrivePolicy,
            //TMPL {{- end }}
            //TMPL {{- if .ReplayPolicy }}
            replayPolicy: args.ReplayPolicy,
            //TMPL {{- end }}
            //TMPL {{- if .SubscriptionRoleArn }}
            subscriptionRoleArn: args.SubscriptionRoleArn,
            //TMPL {{- end }}            
        },
    )
}

function properties(object: aws.sns.TopicSubscription, args: Args) {
    return {
        ID: object.id,
    }
}
