import * as aws from '@pulumi/aws'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'
import * as aws_inputs from '@pulumi/aws/types/input'

interface Args {
    LogGroup: aws.cloudwatch.LogGroup
    Region: pulumi.Output<pulumi.UnwrappedObject<aws.GetRegionResult>>
    Name: string
    Cpu?: string
    Memory?: string
    NetworkMode?: string
    ExecutionRole: aws.iam.Role
    EnvironmentVariables: Record<string, pulumi.Output<string>>
    Image: docker.Image
    PortMappings?: Record<string, object>
    RequiresCompatibilities?: string[]
    EfsVolumes: aws_inputs.ecs.TaskDefinitionVolumeEfsVolumeConfiguration[]
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.TaskDefinition {
    return new aws.ecs.TaskDefinition(args.Name, {
        family: args.Name,
        //TMPL {{- if .Cpu.Raw }}
        cpu: args.Cpu,
        //TMPL {{- end }}
        //TMPL {{- if .Memory.Raw }}
        memory: args.Memory,
        //TMPL {{- end }}
        //TMPL {{- if .NetworkMode.Raw }}
        networkMode: args.NetworkMode,
        //TMPL {{- end }}
        //TMPL {{- if .RequiresCompatibilities.Raw }}
        requiresCompatibilities: args.RequiresCompatibilities,
        //TMPL {{- end }}
        //TMPL {{- if .ExecutionRole.Raw }}
        executionRoleArn: args.ExecutionRole.arn,
        //TMPL {{- end }}
        containerDefinitions: pulumi.jsonStringify([
            {
                name: args.Name,
                image: args.Image.imageName,
                portMappings: args.PortMappings,
                //TMPL {{- if .EnvironmentVariables.Raw }}
                environment: [
                    //TMPL {{- range $name, $value := .EnvironmentVariables.Raw }}
                    //TMPL { name: "{{ $name }}", value: {{ handleIaCValue $value }} },
                    //TMPL {{- end }}
                ],
                //TMPL {{- end }}
                logConfiguration: {
                    logDriver: 'awslogs',
                    options: {
                        'awslogs-group': args.LogGroup.name,
                        'awslogs-region': args.Region.name,
                        'awslogs-stream-prefix': args.Name,
                    },
                },
                //TMPL {{- if .EfsVolumes.Raw }}
                volumes: args.EfsVolumes,
                //TMPL {{- end }}
            },
        ]),
    })
}
