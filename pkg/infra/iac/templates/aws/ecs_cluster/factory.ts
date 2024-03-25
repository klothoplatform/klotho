import * as aws from '@pulumi/aws'
import { ModelCaseWrapper } from '../../wrappers'
import * as awsInputs from '@pulumi/aws/types/input'

interface Args {
    Name: string
    ClusterSettings?: awsInputs.ecs.ClusterSetting[]
    ServiceConnectDefaults: awsInputs.ecs.ClusterServiceConnectDefaults
    Tags: ModelCaseWrapper<Record<string, string>>
    Id?: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.Cluster {
    return new aws.ecs.Cluster(args.Name, {
        //TMPL {{- if .ClusterSettings }}
        settings: args.ClusterSettings,
        //TMPL {{- end }}
        //TMPL {{- if .ServiceConnectDefaults }}
        serviceConnectDefaults: args.ServiceConnectDefaults,
        //TMPL {{- end }}
        //TMPL {{- if .Tags }}
        tags: args.Tags,
        //TMPL {{- end }}
    })
}

function properties(object: aws.ecs.Cluster, args: Args) {
    return {
        Id: object.name,
        UserDataScript: pulumi.interpolate`#!/bin/bash
echo ECS_CLUSTER=${object.name} >> /etc/ecs/ecs.config
`.apply((userData) => Buffer.from(userData).toString('base64')),
    }
}

function importResource(args: Args): aws.ecs.Cluster {
    return aws.ecs.Cluster.get(args.Name, args.Id)
}
