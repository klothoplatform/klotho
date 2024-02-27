import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { OutputInstance } from '@pulumi/pulumi'
import * as awsInputs from '@pulumi/aws/types/input'
import { TemplateWrapper, ModelCaseWrapper } from '../../wrappers'

interface Args {
    AssignPublicIp: Promise<boolean> | OutputInstance<boolean> | boolean
    DeploymentCircuitBreaker:
        | Promise<awsInputs.ecs.ServiceDeploymentCircuitBreaker>
        | OutputInstance<awsInputs.ecs.ServiceDeploymentCircuitBreaker>
        | awsInputs.ecs.ServiceDeploymentCircuitBreaker
    ForceNewDeployment: boolean
    Cluster: aws.ecs.Cluster
    DesiredCount?: number
    LaunchType: string
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnets: aws.ec2.Subnet[]
    TaskDefinition: aws.ecs.TaskDefinition
    Name: string
    LoadBalancers: TemplateWrapper<any[]>
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
    ServiceRegistries: pulumi.Input<awsInputs.ecs.ServiceServiceRegistries>
        Tags: ModelCaseWrapper<Record<string, string>>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.Service {
    return new aws.ecs.Service(
        args.Name,
        {
            launchType: args.LaunchType,
            cluster: args.Cluster.arn,
            //TMPL {{- if .DeploymentCircuitBreaker }}
            //TMPL deploymentCircuitBreaker: {
            //TMPL     enable: {{ .DeploymentCircuitBreaker.Enable }},
            //TMPL     rollback: {{ .DeploymentCircuitBreaker.Rollback }}
            //TMPL },
            //TMPL {{- end }}
            desiredCount: args.DesiredCount,
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
            //TMPL {{- if .Tags }}
            tags: args.Tags,
            //TMPL {{- end }}
        },
        { dependsOn: args.dependsOn }
    )
}
