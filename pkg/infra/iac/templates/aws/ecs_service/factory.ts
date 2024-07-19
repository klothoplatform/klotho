import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { OutputInstance } from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'
import { ModelCaseWrapper, TemplateWrapper } from '../../wrappers'

interface Args {
    AssignPublicIp: Promise<boolean> | OutputInstance<boolean> | boolean
    DeploymentCircuitBreaker: pulumi.Input<awsInputs.ecs.ServiceDeploymentCircuitBreaker>
    DeploymentMaximumPercent: number
    DeploymentMinimumHealthyPercent: number
    EnableExecuteCommand: boolean
    ForceNewDeployment: boolean
    Cluster: aws.ecs.Cluster
    DesiredCount?: number
    LaunchType: string
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnets: aws.ec2.Subnet[]
    TaskDefinition: aws.ecs.TaskDefinition
    Name: string
    HealthCheckGracePeriodSeconds: number
    LoadBalancers: TemplateWrapper<any[]>
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
    ServiceRegistries: pulumi.Input<awsInputs.ecs.ServiceServiceRegistries>
    ServiceConnectConfiguration: pulumi.Input<awsInputs.ecs.ServiceServiceConnectConfiguration>
    CapacityProviderStrategies: pulumi.Input<awsInputs.ecs.ServiceCapacityProviderStrategy[]>
    Tags: ModelCaseWrapper<Record<string, string>>
    Arn: string
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.Service {
    return new aws.ecs.Service(
        args.Name,
        {
            //TMPL {{- if .LaunchType }}
            launchType: args.LaunchType,
            //TMPL {{- end }}
            //TMPL {{- if .CapacityProviderStrategies }}
            capacityProviderStrategies: args.CapacityProviderStrategies,
            //TMPL {{- end }}
            cluster: args.Cluster.arn,
            //TMPL {{- if .DeploymentCircuitBreaker }}
            //TMPL deploymentCircuitBreaker: {
            //TMPL     enable: {{ .DeploymentCircuitBreaker.Enable }},
            //TMPL     rollback: {{ .DeploymentCircuitBreaker.Rollback }}
            //TMPL },
            //TMPL {{- end }}
            //TMPL {{- if .DeploymentMaximumPercent }}
            deploymentMaximumPercent: args.DeploymentMaximumPercent,
            //TMPL {{- end }}
            //TMPL {{- if (ne .DeploymentMinimumHealthyPercent nil ) }}
            deploymentMinimumHealthyPercent: args.DeploymentMinimumHealthyPercent,
            //TMPL {{- end }}
            desiredCount: args.DesiredCount,
            //TMPL {{- if .EnableExecuteCommand }}
            enableExecuteCommand: args.EnableExecuteCommand,
            //TMPL {{- end }}
            //TMPL {{- if .HealthCheckGracePeriodSeconds }}
            healthCheckGracePeriodSeconds: args.HealthCheckGracePeriodSeconds,
            //TMPL {{- end }}
            forceNewDeployment: args.ForceNewDeployment,
            //TMPL {{- if .LoadBalancers }}
            loadBalancers: args.LoadBalancers,
            //TMPL {{- end }}
            //TMPL {{- if or .SecurityGroups .Subnets .AssignPublicIp }}
            networkConfiguration: {
                //TMPL {{- if .AssignPublicIp }}
                assignPublicIp: args.AssignPublicIp,
                //TMPL {{- end }}
                //TMPL {{- if .Subnets }}
                subnets: args.Subnets.map((sn) => sn.id),
                //TMPL {{- end }}
                //TMPL {{- if .SecurityGroups }}
                securityGroups: args.SecurityGroups.map((sg) => sg.id),
                //TMPL {{- end }}
            },
            //TMPL {{- end }}
            taskDefinition: args.TaskDefinition.arn,
            waitForSteadyState: true,
            //TMPL {{- if .ServiceRegistries }}
            serviceRegistries: args.ServiceRegistries,
            //TMPL {{- end }}
            //TMPL {{- if .ServiceConnectConfiguration }}
            serviceConnectConfiguration: args.ServiceConnectConfiguration,
            //TMPL {{- end }}
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        { dependsOn: args.dependsOn }
    )
}

function properties(object: aws.ecs.Service, args: Args) {
    return {
        // We should replace Arn with Id in the future
        Arn: object.id,
        Name: object.name,
    }
}

function importResource(args: Args): aws.ecs.Service {
    // Imported ID must be in the form cluster-name/service-name.
    // The Id field is the service's ARN, which includes cluster-name/service-name as a suffix.
    return aws.ecs.Service.get(args.Name, args.Arn.split('/').slice(-2).join('/'))
}
