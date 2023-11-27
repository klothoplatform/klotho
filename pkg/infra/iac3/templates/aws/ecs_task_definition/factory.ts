import * as aws from '@pulumi/aws'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'
import { TemplateWrapper } from '../../wrappers'

interface Args {
    LogGroup: aws.cloudwatch.LogGroup
    Region: pulumi.Output<pulumi.UnwrappedObject<aws.GetRegionResult>>
    Name: string
    Cpu?: string
    Memory?: string
    NetworkMode?: string
    ExecutionRole: aws.iam.Role
    TaskRole: aws.iam.Role
    EnvironmentVariables: TemplateWrapper<Record<string, pulumi.Output<string>>>
    Image: docker.Image
    PortMappings?: Record<string, object>
    RequiresCompatibilities?: string[]
    EfsVolumes: TemplateWrapper<awsInputs.ecs.TaskDefinitionVolumeEfsVolumeConfiguration[]>
    MountPoints: Record<string,string>
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
        containerDefinitions: pulumi.jsonStringify([
            {
                name: args.Name,
                image: args.Image.imageName,
                portMappings: args.PortMappings,
                //TMPL {{- if .EnvironmentVariables }}
                environment: args.EnvironmentVariables,
                //TMPL {{- end }}
                //TMPL {{- if .MountPoints }}
                mountPoints: args.MountPoints,
                //TMPL {{- end }}
                logConfiguration: {
                    logDriver: 'awslogs',
                    options: {
                        'awslogs-group': args.LogGroup.name,
                        'awslogs-region': args.Region.name,
                        'awslogs-stream-prefix': args.Name,
                    },
                },
            },
        ]),
    })
}
