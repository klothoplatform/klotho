import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'
import * as awsInputs from '@pulumi/aws/types/input'

interface Args {
    Name: string
    Cluster: string
    CapacityProviders: string[]
    DefaultCapacityProviderStrategy: awsInputs.ecs.ClusterCapacityProvidersDefaultCapacityProviderStrategy[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.ClusterCapacityProviders {
    return new aws.ecs.ClusterCapacityProviders(args.Name, {
        clusterName: args.Cluster,
        capacityProviders: args.CapacityProviders,
        defaultCapacityProviderStrategies: args.DefaultCapacityProviderStrategy,
    })
}

function properties(object: aws.ecs.ClusterCapacityProviders, args: Args) {
    return {
        Id: object.id,
    }
}
