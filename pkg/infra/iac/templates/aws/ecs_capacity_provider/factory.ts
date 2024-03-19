import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'
import * as awsInputs from '@pulumi/aws/types/input'

interface Args {
    Name: string
    AutoScalingGroupProvider: awsInputs.ecs.CapacityProviderAutoScalingGroupProvider
    Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.CapacityProvider {
    return new aws.ecs.CapacityProvider(args.Name, {
        autoScalingGroupProvider: args.AutoScalingGroupProvider,
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ecs.CapacityProvider, args: Args) {
    return {
        Arn: object.arn,
        Id: object.name,
    }
}

function importResource(args: Args): aws.ecs.CapacityProvider {
    return aws.ecs.CapacityProvider.get(args.Name, args.Id)
}
