import * as aws from '@pulumi/aws'
import * as docker from '@pulumi/docker'
import * as pulumi from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'

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
    EfsVolumes: awsInputs.ecs.TaskDefinitionVolumeEfsVolumeConfiguration[]
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
        containerDefinitions: pulumi.jsonStringify([
            {
                name: args.Name,
                image: args.Image.imageName,
                portMappings: args.PortMappings,
                //TMPL {{- if .EnvironmentVariables }}
                environment: [
                    //TMPL {{- range $name, $value := .EnvironmentVariables }}
                    //TMPL { name: "{{ $name }}", value: {{ $value }} },
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
                //TMPL {{- if .EfsVolumes }}
                volumes: args.EfsVolumes,
                //TMPL {{- end }}
            },
        ]),
    })
}
