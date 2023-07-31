import * as aws from '@pulumi/aws'
import * as pulumi from '@pulumi/pulumi'
import { OutputInstance } from '@pulumi/pulumi'
import * as aws_input from '@pulumi/aws/types/input'

interface Args {
    AssignPublicIp: Promise<boolean> | OutputInstance<boolean> | boolean
    DeploymentCircuitBreaker:
        | Promise<aws_input.ecs.ServiceDeploymentCircuitBreaker>
        | OutputInstance<aws_input.ecs.ServiceDeploymentCircuitBreaker>
        | aws_input.ecs.ServiceDeploymentCircuitBreaker
    ForceNewDeployment: boolean
    Cluster: aws.ecs.Cluster
    DesiredCount?: number
    LaunchType: string
    SecurityGroups: aws.ec2.SecurityGroup[]
    Subnets: aws.ec2.Subnet[]
    TaskDefinition: aws.ecs.TaskDefinition
    Name: string
    LoadBalancers: any[]
    dependsOn?: pulumi.Input<pulumi.Input<pulumi.Resource>[]> | pulumi.Input<pulumi.Resource>
}

// noinspection JSUnusedLocalSymbols
function create(args: Args): aws.ecs.Service {
    return new aws.ecs.Service(
        args.Name,
        {
            launchType: args.LaunchType,
            cluster: args.Cluster.arn,
            //TMPL {{- if .DeploymentCircuitBreaker.Raw }}
            //TMPL deploymentCircuitBreaker: {
            //TMPL     enable: {{ .DeploymentCircuitBreaker.Raw.Enable }},
            //TMPL     rollback: {{ .DeploymentCircuitBreaker.Raw.Rollback }}
            //TMPL },
            //TMPL {{- end }}
            desiredCount: args.DesiredCount,
            forceNewDeployment: args.ForceNewDeployment,
            //TMPL {{- if .LoadBalancers.Raw }}
            loadBalancers: args.LoadBalancers,
            //TMPL {{- end }}
            //TMPL {{- if or .SecurityGroups.Raw .Subnets.Raw .AssignPublicIp.Raw }}
            networkConfiguration: {
                //TMPL {{- if .AssignPublicIp.Raw }}
                assignPublicIp: args.AssignPublicIp,
                //TMPL {{- end }}
                //TMPL {{- if .Subnets.Raw }}
                subnets: args.Subnets.map((sn) => sn.id),
                //TMPL {{- end }}
                //TMPL {{- if .SecurityGroups.Raw }}
                securityGroups: args.SecurityGroups.map((sg) => sg.id),
                //TMPL {{- end }}
            },
            //TMPL {{- end }}
            taskDefinition: args.TaskDefinition.arn,
        },
        { dependsOn: args.dependsOn }
    )
}
