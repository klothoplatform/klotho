import * as aws from '@pulumi/aws'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'
import { TemplateWrapper } from '../../wrappers'

interface Args {
    Name: string
    NetworkMode?: string
    ExecutionRole: aws.iam.Role
    TaskRole: aws.iam.Role
    RequiresCompatibilities?: string[]
    EfsVolumes: TemplateWrapper<awsInputs.ecs.TaskDefinitionVolumeEfsVolumeConfiguration[]>
    ContainerDefinitions: TemplateWrapper<awsInputs.ecs.TaskDefinitionContainerDefinitions[]>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.TaskDefinition {
    return new aws.ecs.TaskDefinition(args.Name, {
        family: args.Name,
        //TMPL {{- if .Cpu }}
        cpu: args.Cpu,
        //TMPL {{- end }}
        //TMPL {{- if .Memory }}
        memory: args.Memory,
        //TMPL {{- end }}
        //TMPL {{- if .NetworkMode }}
        networkMode: args.NetworkMode,
        //TMPL {{- end }}
        //TMPL {{- if .RequiresCompatibilities }}
        requiresCompatibilities: args.RequiresCompatibilities,
        //TMPL {{- end }}
        executionRoleArn: args.ExecutionRole.arn,
        //TMPL {{- if .TaskRole }}
        taskRoleArn: args.TaskRole.arn,
        //TMPL {{- end }}
        //TMPL {{- if .EfsVolumes }}
        volumes: args.EfsVolumes,
        //TMPL {{- end }}
        containerDefinitions: pulumi.jsonStringify(args.ContainerDefinitions),
    })
}
